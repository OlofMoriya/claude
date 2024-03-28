package services

import (
	"bufio"
	"claude/data"
	"claude/models"
	"fmt"
	"io"
	"net/http"
)

func AwaitedQuery(prompt string, model models.Model, historyRepository data.HistoryRepository, historyCount int, contextId int64) {

	history, err := historyRepository.GetHistoryByContextId(contextId, historyCount)

	req := model.CreateRequest(contextId, prompt, false, history)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Errorf("failed to execute request: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if err != nil {
			panic("failed to read response body on non-OK status")
		}
		fmt.Printf("\nresp: %v", resp)
		fmt.Printf("\nerr: %v", err)
		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	fmt.Println(string(bodyBytes))
	if err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error reading response body: %v\n", err)
	} // Close the response body when done
	defer resp.Body.Close()

	model.HandleBodyBytes(bodyBytes)
	//TODO: Handle token use
}

func StreamedQuery(prompt string, model models.Model, historyRepository data.HistoryRepository, historyCount int, contextId int64) {
	history, err := historyRepository.GetHistoryByContextId(contextId, historyCount)

	//fmt.Printf("%+v\n", history)
	if err != nil {
		panic(fmt.Sprintf("Could not fetch history %s", err))
	}

	req := model.CreateRequest(contextId, prompt, true, history)

	// fmt.Printf("%+v\n", req)
	// panic("Stop for testing")
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
		fmt.Printf("resp:%v", resp)
		fmt.Printf("body:%v", resp.Body)

		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	reader := bufio.NewReader(resp.Body)
	finished := false
	for !finished {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// fmt.Println("failed to read bytes from stream response")
			finished = true
			continue
		}

		model.HandleStreamedLine(line)
	}
}
