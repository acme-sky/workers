package handlers

import (
	"context"
	"sort"
	"time"

	"github.com/charmbracelet/log"

	pb "github.com/acme-sky/geodistance-api/pkg/distance/proto"
	"github.com/acme-sky/workers/internal/config"
	"github.com/acme-sky/workers/internal/db"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Service task used to sort all rent services with key the distance between
// user and rent geolocalizations.
func STSortRentServices(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	db, _ := db.GetDb()
	var offer models.Offer

	if err := db.Where("id = ?", int(variables["offer_id"].(float64))).Preload("User").Preload("Journey").Preload("Journey.Flight1").First(&offer).Error; err != nil {
		log.Errorf("[%s] [%d] Error on getting offer %s", job.Type, jobKey, err.Error())
	}

	if offer.User.Address == nil {
		log.Warnf("[%s] [%d] User does not have an address", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	conf, _ := config.GetConfig()

	conn, err := grpc.Dial(conf.String("geodistance.api"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("[%s] [%d] Can't connect to Geodistance url: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	defer conn.Close()
	c := pb.NewDistanceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	userGeometry, err := c.FindGeometry(ctx, &pb.AddressRequest{
		Address: *offer.User.Address,
	})
	if err != nil {
		log.Errorf("[%s] [%d] Can't find geometry for user: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	var rents []models.Rent
	if err := db.Find(&rents).Error; err != nil {
		log.Errorf("[%s] [%d] Rents not found %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	type RentDistance struct {
		Id       uint
		Distance int
	}
	var distances []RentDistance

	for _, rent := range rents {
		distance, err := c.FindDistance(ctx, &pb.DistanceRequest{
			Origin:      &pb.MapPosition{Latitude: userGeometry.Latitude, Longitude: userGeometry.Longitude},
			Destination: &pb.MapPosition{Latitude: rent.Latitude, Longitude: rent.Longitude},
		})
		if err != nil {
			log.Warnf("[%s] [%d] Can't find distance for %s: %s", job.Type, jobKey, rent.Name, err.Error())
			distances = append(distances, RentDistance{Id: rent.Id, Distance: 999999})
			continue
		}
		distances = append(distances, RentDistance{Id: rent.Id, Distance: int(distance.GetDistance())})
	}

	if len(rents) == 0 {
		log.Errorf("[%s] [%d] There is no available rent company: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	sort.Slice(distances, func(i, j int) bool {
		return distances[i].Distance > distances[j].Distance
	})

	variables["rent_companies"] = distances
	variables["rent_status"] = "No"

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	ctx = context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)
	acmejob.JobVariables[job.Type] <- variables

	acmejob.JobStatuses.Close(job.Type, 0)
}
