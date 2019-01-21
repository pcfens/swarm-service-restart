package main

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	cron "github.com/pcfens/swarm-service-restart/cron"
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
	crontab.AddFunc("@every 15s", func(unused string) { refreshJobs(crontab, &crontabMap) }, "")
	crontab.Start()
	select {}
}

func removeStringFromArray(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
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

	// Build a list of services we don't check so we can remove them from
	// the crontab
	var uncheckedServices []string
	for serviceID := range *crontabMap {
		uncheckedServices = append(uncheckedServices, serviceID)
	}

	for _, service := range services {
		cronValue := service.Spec.Labels["edu.wm.restartService.schedule"]

		// If we find the serviceID in the list then remove it from our list
		// of unchecked services
		for i := 0; i < len(uncheckedServices); i++ {
			if uncheckedServices[i] == service.ID {
				uncheckedServices = removeStringFromArray(uncheckedServices, i)
			}
		}

		// If the service has a cronjob already scheduled, check the timing
		if val, ok := (*crontabMap)[service.ID]; ok {
			// If the timing isn't right, unschedule it and set a new time
			if val.schedule != cronValue {
				crontab.Remove(val.entry)
				crontabID, err := crontab.AddFunc(cronValue, restartService, service.ID)
				fmt.Printf("Rescheduled %s to restart at %s\n", service.ID, cronValue)
				if err != nil {
					panic(err)
				}

				// Update the map/struct with the new time
				(*crontabMap)[service.ID] = cronEntry{schedule: cronValue, entry: crontabID}
			}
		} else {
			crontabID, err := crontab.AddFunc(cronValue, restartService, service.ID)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Scheduled %s to restart at %s\n", service.ID, cronValue)
			(*crontabMap)[service.ID] = cronEntry{schedule: cronValue, entry: crontabID}
		}
	}

	// Everything that's left unchecked should be removed from the crontab
	for i, serviceID := range uncheckedServices {
		if val, ok := (*crontabMap)[uncheckedServices[i]]; ok {
			crontab.Remove(val.entry)      // Remove the entry from the crontab
			delete(*crontabMap, serviceID) // Remove the entry from our easier to parse map
			fmt.Printf("Removed %s from restart schedule\n", serviceID)
		}
	}

	return
}

func restartService(ServiceID string) {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	filters := filters.NewArgs()
	filters.Add("id", ServiceID)
	filters.Add("label", "edu.wm.restartService.schedule")

	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{Filters: filters})
	if err != nil {
		panic(err)
	}

	if len(services) == 0 {
		fmt.Printf("Skipping restart of %s because it's no longer marked for removal\n", ServiceID)
		return
	}

	fmt.Printf("Restarting %s\n", ServiceID)

	service, _, err := cli.ServiceInspectWithRaw(context.Background(), ServiceID, types.ServiceInspectOptions{InsertDefaults: false})
	if err != nil {
		panic(err)
	}

	currentSpec := service.Spec
	currentSpec.TaskTemplate.ForceUpdate++

	_, err = cli.ServiceUpdate(context.Background(), ServiceID, service.Meta.Version, currentSpec, types.ServiceUpdateOptions{QueryRegistry: false})
	if err != nil {
		panic(err)
	}

	return
}
