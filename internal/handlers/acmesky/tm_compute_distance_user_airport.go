package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	pb "github.com/acme-sky/geodistance-api/pkg/distance/proto"
	"github.com/acme-sky/workers/internal/config"
	"github.com/acme-sky/workers/internal/db"
	"github.com/acme-sky/workers/internal/http"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Task used to find distance between departaure airport and user.
func TMComputeDistanceUserAirport(client worker.JobClient, job entities.Job) {
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

	endpoint := fmt.Sprintf("%s/airports/code/%s/", offer.Journey.Flight1.Airline, offer.Journey.Flight1.DepartaureAirport)
	airport, err := http.GetAirportInfo(endpoint)
	if err != nil {
		log.Errorf("[%s] [%d] Can't find info for departaure airport: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}

	distance, err := c.FindDistance(ctx, &pb.DistanceRequest{
		Origin:      &pb.MapPosition{Latitude: userGeometry.Latitude, Longitude: userGeometry.Longitude},
		Destination: &pb.MapPosition{Latitude: airport.Latitude, Longitude: airport.Longitude},
	})
	if err != nil {
		log.Errorf("[%s] [%d] Can't find distance: %s", job.Type, jobKey, err.Error())
		acmejob.FailJob(client, job)
		return
	}
	variables["distance"] = distance.GetDistance() / 1000
	log.Info("[%s] [%d] Found a distance of: %d km", job.Type, jobKey, variables["distance"])

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)
	acmejob.JobVariables[job.Type] <- variables

	acmejob.JobStatuses.Close(job.Type, 0)
}
