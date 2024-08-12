package tools

import (
	"Mars/auth"
	"Mars/icarus"
	"context"
	"fmt"
	"time"
)

func BasicControl() {
	icarusClient := auth.CreateIcarusClient()

	req := &icarus.GetAllNavStatusRequest{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dronesStatus, err := icarusClient.Get_All_Nav_Status(ctx, req)
	if err != nil {
		fmt.Println(err)
		fmt.Println("womp womp")
	} else {
		for _, drone := range dronesStatus.GetStatus() {
			payloadReq := &icarus.GetPayloadsRequest{EntityId: drone.GetEntityId()}
			payloadStatus, err := icarusClient.Get_Payloads(ctx, payloadReq)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Println(payloadStatus)
			fmt.Println()
		}
	}
}

/*func HangarHandleDrone(airBaseId, droneId uint64, load_option string, client icarus.IcarusClient) error {
	//
	droneNameFull, err := getNameFromId(droneId, client)
	if err != nil {
		fmt.Println(err)
	}
	if droneNameFull == "" {
		return errors.New("Drone not found")
	}
	splitName := strings.Split(droneNameFull, "-")
	droneName := splitName[0]

	//Get all payloads
	resp, err := getPayloadsStatus(airBaseId, client)
	if err != nil {
		fmt.Println("error")
	}
	allPayloadsArray := resp.GetPayloadStatus()
	sort.Slice(allPayloadsArray, func(i, j int) bool {
		return allPayloadsArray[i].Name < allPayloadsArray[j].Name
	})
	HangarPayloadId := uint32(999)
	index := 0
	for _, payloadStatus := range allPayloadsArray {
		if strings.Contains(payloadStatus.GetName(), droneName) {
			HangarPayloadId = uint32(index)
			fmt.Println(payloadStatus.GetName())
			fmt.Println(HangarPayloadId)
		}
		index++
	}
	fmt.Println("HangarPayloadId", HangarPayloadId)

	//Create load/unload from hangar command
	hangarControllerCommand := &icarus.PayloadCommand{}
	if load_option == "load" {
		hangarControllerCommand = &icarus.PayloadCommand{
			Name:        "load",
			Description: "Load an entity with the appropriate class into this hangar",
			Args: []*icarus.PayloadArguments{
				{
					Name:        "Load ID",
					Description: "An entity ID to load",
					Type:        icarus.PayloadArgType_PayloadArgType_UINT,
					Value:       fmt.Sprint(droneId),
				},
			},
		}
	} else {
		hangarControllerCommand = &icarus.PayloadCommand{
			Name:        "unload",
			Description: "Unload an entity with the appropriate class out of this hangar",
			Args: []*icarus.PayloadArguments{
				{
					Name:        "Unload ID",
					Description: "An entity ID to unload",
					Type:        icarus.PayloadArgType_PayloadArgType_UINT,
					Value:       fmt.Sprint(droneId),
				},
			},
		}

	}

	fmt.Println(hangarControllerCommand)

	req := &icarus.ExecutePayloadCommandRequest{
		EntityId:  airBaseId,
		PayloadId: HangarPayloadId,
		Command:   hangarControllerCommand,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp2, err := client.Execute_Payload_Command(ctx, req)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(resp2)

	//Find the payload command that says launch
	return nil
}*/

func getNameFromId(entityId uint64, client icarus.IcarusClient) (string, error) {
	// Create an instance of the DroneStatusRequest struct
	// Pass the context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	navStatus := &icarus.GetNavStatusRequest{
		EntityId: entityId,
	}
	resp, err := client.Get_Nav_Status(ctx, navStatus)
	if err != nil {
		// If there is an error, return a new error
		return "", err
	}
	return resp.GetStatus().Name, nil

}
