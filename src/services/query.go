package services

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"owl/data"
	"owl/logger"
	"owl/models"
)

func AwaitedQuery(prompt string, model models.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, image bool, pdf string) {
	history, err := historyRepository.GetHistoryByContextId(context.Id, historyCount)
	if err != nil {
		log.Println("error while fetching history for context", err)
	}

	req := model.CreateRequest(context, prompt, false, history, image, pdf)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("failed to execute request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		println(fmt.Sprintf("\nresp: %v", resp))
		println(fmt.Sprintf("\n\body: %v\n\n", resp.Body))
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(bytes))
		println(fmt.Sprintf("\nerr: %v", err))

		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	logger.Debug.Printf("statusCode: %s", resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debug.Println(err)
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error reading response body: %v\n", err))
	} // Close the response body when done
	defer resp.Body.Close()

	logger.Debug.Println("Received a response without streaming")
	logger.Debug.Printf("bodyBytes %s", string(bodyBytes))
	model.HandleBodyBytes(bodyBytes)
	//TODO: Handle token use
}

func StreamedQuery(prompt string, model models.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, image bool, pdf string) {
	history, err := historyRepository.GetHistoryByContextId(context.Id, historyCount)

	if err != nil {
		panic(fmt.Sprintf("Could not fetch history %s", err))
	}

	req := model.CreateRequest(context, prompt, true, history, image, pdf)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("Failed to execute request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if err != nil {
			panic("Failed to read response body on non-OK status")
		}

		bytes, err := io.ReadAll(resp.Body)
		println("bytes", string(bytes), err)

		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	reader := bufio.NewReader(resp.Body)
	finished := false
	for !finished {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("failed to read bytes from stream response")
			finished = true
			continue
		}

		model.HandleStreamedLine(line)
	}
}
