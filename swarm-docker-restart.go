package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	cron "gopkg.in/robfig/cron.v2"
)

type cronEntry struct {
	schedule string
	entry    cron.EntryID
}

type cronMap map[string]cronEntry

func main() {
	var crontabMap cronMap
	crontabMap = make(cronMap)

	crontab := cron.New()
	crontab.AddFunc("@every 15s", func() { refreshJobs(crontab, &crontabMap) })
	crontab.Start()
	select {}
}

func refreshJobs(crontab *cron.Cron, crontabMap *cronMap) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	filters := filters.NewArgs()
	filters.Add("label", "edu.wm.restartService.schedule")

	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})
	if err != nil {
		panic(err)
	}

	for _, service := range services {
		cronValue := service.Spec.Labels["edu.wm.restartService.schedule"]

		// If the service has a cronjob already scheduled, check the timing
		if val, ok := (*crontabMap)[service.ID]; ok {
			// If the timing isn't right, unschedule it and set a new time
			if val.schedule != cronValue {
				crontab.Remove(val.entry)
				crontabID, err := crontab.AddFunc(cronValue, func() { restartService(service.ID) })
				fmt.Printf("Rescheduled %s to restart at %s\n", service.ID, cronValue)
				if err != nil {
					panic(err)
				}

				// Update the map/struct with the new time
				(*crontabMap)[service.ID] = cronEntry{schedule: cronValue, entry: crontabID}
			}
		} else {
			crontabID, err := crontab.AddFunc(cronValue, func() { restartService(service.ID) })
			if err != nil {
				panic(err)
			}
			fmt.Printf("Scheduled %s to restart at %s\n", service.ID, cronValue)
			(*crontabMap)[service.ID] = cronEntry{schedule: cronValue, entry: crontabID}
		}
	}
}

func restartService(ServiceID string) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Restarting %s\n", ServiceID)

	service, _, err := cli.ServiceInspectWithRaw(context.Background(), ServiceID, types.ServiceInspectOptions{InsertDefaults: false})
	if err != nil {
		panic(err)
	}

	currentSpec := service.Spec
	currentSpec.TaskTemplate.ForceUpdate = 1

	_, err = cli.ServiceUpdate(context.Background(), ServiceID, service.Meta.Version, currentSpec, types.ServiceUpdateOptions{QueryRegistry: false})
	if err != nil {
		panic(err)
	}

	return
}
