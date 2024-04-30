package handlers

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/acme-sky/workers/internal/db"
	acmejob "github.com/acme-sky/workers/internal/job"
	"github.com/acme-sky/workers/internal/models"
	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
)

// Service Task raised by ACMESky Interests Manager lame in a sequential loop
// for available flights.
// Create a new offer from an available flight and then send the offer via
// Prontogram.
func STPrepareOffer(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	journeys := variables["journeys"].([]interface{})
	index := int(variables["loopCounter"].(float64)) - 1

	db, _ := db.GetDb()
	var journey models.Journey

	if err := db.Where("id = ?", int(journeys[index].(float64))).Preload("Flight1").Preload("Flight2").Preload("User").First(&journey).Error; err != nil {
		log.Errorf("[%s] [%d] Journey not found", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	}

	body := models.OfferInput{
		Flight1: models.OfferInputFields{
			DepartaureAirport: journey.Flight1.DepartaureAirport,
			ArrivalAirport:    journey.Flight1.ArrivalAirport,
			DepartaureTime:    journey.Flight1.DepartaureTime.Format("02/01/2006 15:04"),
			ArrivalTime:       journey.Flight1.ArrivalTime.Format("02/01/2006 15:04"),
			Cost:              journey.Flight1.Cost,
		},
		JourneyId: int(journey.Id),
		UserId:    journey.UserId,
		Name:      journey.User.Name,
	}

	if journey.Flight2 != nil {
		body.Flight2 = &models.OfferInputFields{
			DepartaureAirport: journey.Flight2.DepartaureAirport,
			ArrivalAirport:    journey.Flight2.ArrivalAirport,
			DepartaureTime:    journey.Flight2.DepartaureTime.Format("02/01/2006 15:04"),
			ArrivalTime:       journey.Flight2.ArrivalTime.Format("02/01/2006 15:04"),
			Cost:              journey.Flight2.Cost,
		}
	}

	offer := models.NewOffer(body)

	if created := db.Create(&offer); created == nil {
		log.Errorf("[%s] [%d] Offer not saved", job.Type, jobKey)
		acmejob.FailJob(client, job)
		return
	} else {
		log.Infof("[%s] [%d] Offer saved", job.Type, jobKey)
		var flightInstance models.AvailableFlight
		if err := db.Where("id = ?", journey.Flight1Id).First(&flightInstance).Error; err != nil {
			log.Errorf("[%s] [%d] Error on getting flight %s", job.Type, jobKey, err.Error())
		}
		flightInstance.OfferSent = true
		if err := db.Save(&flightInstance).Error; err != nil {
			log.Errorf("[%s] [%d] Error on saving flight %s", job.Type, jobKey, err.Error())
		}

		if journey.Flight2 != nil {
			var flightInstance models.AvailableFlight
			if err := db.Where("id = ?", journey.Flight2Id).First(&flightInstance).Error; err != nil {
				log.Errorf("[%s] [%d] Error on getting flight %s", job.Type, jobKey, err.Error())
			}
			flightInstance.OfferSent = true
			if err := db.Save(&flightInstance).Error; err != nil {
				log.Errorf("[%s] [%d] Error on saving flight %s", job.Type, jobKey, err.Error())
			}
		}
	}

	variables["offer"] = offer

	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}
	ctx := context.Background()
	_, err = request.Send(ctx)
	if err != nil {
		acmejob.FailJob(client, job)
		return
	}

	log.Infof("[%s] [%d] Successfully completed job", job.Type, jobKey)
	acmejob.JobStatuses.Close(job.Type, 0)
}
