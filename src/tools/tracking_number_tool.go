package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
)

type TrackingNumberTool struct {
}

type TrackingNumberLookupInput struct {
	TrackingNumber string
}

func (tool *TrackingNumberTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *TrackingNumberTool) Run(i map[string]string) (string, error) {

	fmt.Printf("\n\ncalled run on tracking number tool. Input %v \n\n", i)

	trackingnumber, exists := i["TrackingNumber"]

	if !exists {
		fmt.Println("parsing error")
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	out, err := exec.Command("login-and-status.sh", trackingnumber).Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value, nil
}

func (tool *TrackingNumberTool) GetName() string {
	return "early_bird_track_lookup"
}

func (tool *TrackingNumberTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches status from a shipment trackingnumber in the early bird logistics chain. Can help anwser questions about where a delivery is in the process or if anything went wrong with the shipment.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"TrackingNumber": {
					Type:        "string",
					Description: "TrackingNumber that can be used to look up the order.",
				},
			},
		},
	}
}

func init() {
	Register(&TrackingNumberTool{})
}
