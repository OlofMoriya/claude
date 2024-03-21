package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Model interface {
	createRequest(contextId int64, prompt string, streaming bool, history []History) *http.Request
	handleStreamedLine(line []byte)
	handleBodyBytes(bytes []byte)
}

func awaitedQuery(prompt string, model Model) {
	//TODO: context id
	req := model.createRequest(0, prompt, false, []History{})

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
		panic(fmt.Errorf("received non-OK response status: %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error reading response body: %v\n", err)
	} // Close the response body when done
	defer resp.Body.Close()

	model.handleBodyBytes(bodyBytes)
	//TODO: Handle token use
}

func streamedQuery(prompt string, model Model, historyRepository HistoryRepository, historyCount int, contextId int64) {
	//TODO: context id
	history, err := historyRepository.getHistoryByContextId(contextId, historyCount)

	//fmt.Printf("%+v\n", history)
	if err != nil {
		panic(fmt.Sprintf("Could not fetch history %s", err))
	}

	req := model.createRequest(contextId, prompt, true, history)

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

		model.handleStreamedLine(line)
	}
}

func main() {

	user := User{Id: "olof", Name: "olof"}

	prompt := ""
	var context_id int64 = 0
	historyCount := 0

	if len(os.Args) > 1 {
		prompt = os.Args[len(os.Args)-1]
	} else {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Prompt:")
		prompt, _ = reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
	}

	if len(os.Args) > 2 {
		count, err := strconv.Atoi(os.Args[len(os.Args)-2])
		if err != nil {
			panic(err)
		}
		historyCount = count
	}

	if len(os.Args) > 3 {
		context_name := os.Args[len(os.Args)-3]
		context, _ := user.getContextByName(context_name)

		if context == nil {
			new_context := Context{Name: context_name}
			id, err := user.insertContext(new_context)
			if err != nil {
				panic(fmt.Sprintf("Could not create a new context with name %s, %s", context_name, err))
			}
			context_id = id
		} else {
			context_id = context.Id
		}
		fmt.Printf("found a context_id: %d for context %v", context_id, context)
	}

	stream := true

	//TODO Model selector
	cliResponseHandler := CliResponseHandler{Repository: user}
	var claudeModel Model = &ClaudeModel{responseHandler: cliResponseHandler}

	if stream {
		streamedQuery(prompt, claudeModel, user, historyCount, context_id)
	} else {
		awaitedQuery(prompt, claudeModel)
	}
}
