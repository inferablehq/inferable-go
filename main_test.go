package inferable

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

type EchoInput struct {
	Input string
}

func echo(input EchoInput) string {
	return input.Input
}

type ReverseInput struct {
	Input string
}

func reverse(input ReverseInput) string {
	runes := []rune(input.Input)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func TestInferableFunctions(t *testing.T) {
	if os.Getenv("INFERABLE_API_SECRET") == "" {
		err := godotenv.Load("./.env")

		if err != nil {
			panic(err)
		}
	}

	apiEndpoint := os.Getenv("INFERABLE_API_ENDPOINT")
	apiSecret := os.Getenv("INFERABLE_API_SECRET")

	if apiEndpoint == "" || apiSecret == "" {
		t.Fatal("INFERABLE_API_ENDPOINT or INFERABLE_API_SECRET not set in .env file")
	}

	inferableInstance, err := New(InferableOptions{
		APIEndpoint: apiEndpoint,
		APISecret:   apiSecret,
	})
	if err != nil {
		t.Fatalf("Error creating Inferable instance: %v", err)
	}

	service, err := inferableInstance.RegisterService("string_operations")
	if err != nil {
		t.Fatalf("Error registering service: %v", err)
	}

	err = service.RegisterFunc(Function{
		Func:        echo,
		Description: "Echoes the input string",
		Name:        "echo",
	})
	if err != nil {
		t.Fatalf("Error registering echo function: %v", err)
	}

	err = service.RegisterFunc(Function{
		Func:        reverse,
		Description: "Reverses the input string",
		Name:        "reverse",
	})
	if err != nil {
		t.Fatalf("Error registering reverse function: %v", err)
	}

	jsonDef, err := inferableInstance.ToJSONDefinition()
	if err != nil {
		t.Fatalf("Error generating JSON definition: %v", err)
	}
	t.Logf("JSON Definition:\n%s\n", string(jsonDef))

	t.Run("Echo Function", func(t *testing.T) {
		testInput := EchoInput{Input: "Hello, Inferable!"}
		result, err := inferableInstance.CallFunc("string_operations", "echo", testInput)
		if err != nil {
			t.Fatalf("Error calling echo function: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 return value, got %d", len(result))
		}

		returnedString := result[0].Interface().(string)
		if returnedString != testInput.Input {
			t.Errorf("Echo function returned incorrect result. Expected: %s, Got: %s", testInput.Input, returnedString)
		}
	})

	t.Run("Reverse Function", func(t *testing.T) {
		testInput := ReverseInput{Input: "Hello, Inferable!"}
		result, err := inferableInstance.CallFunc("string_operations", "reverse", testInput)
		if err != nil {
			t.Fatalf("Error calling reverse function: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 return value, got %d", len(result))
		}

		returnedString := result[0].Interface().(string)
		if returnedString != "!elbarefnI ,olleH" {
			t.Errorf("Reverse function returned incorrect result. Expected: %s, Got: %s", testInput.Input, returnedString)
		}
	})

	t.Run("Server Health Check", func(t *testing.T) {
		err := inferableInstance.ServerOk()
		if err != nil {
			t.Fatalf("Server health check failed: %v", err)
		}
		t.Log("Server health check passed")
	})

	t.Run("Machine ID Generation", func(t *testing.T) {
		machineID := inferableInstance.GetMachineID()
		if machineID == "" {
			t.Error("Machine ID is empty")
		}
		t.Logf("Generated Machine ID: %s", machineID)
	})

	t.Run("Machine ID Consistency", func(t *testing.T) {
		instance1, err := New(InferableOptions{
			APIEndpoint: apiEndpoint,
			APISecret:   apiSecret,
		})
		if err != nil {
			t.Fatalf("Error creating first Inferable instance: %v", err)
		}
		id1 := instance1.GetMachineID()

		instance2, err := New(InferableOptions{
			APIEndpoint: apiEndpoint,
			APISecret:   apiSecret,
		})
		if err != nil {
			t.Fatalf("Error creating second Inferable instance: %v", err)
		}
		id2 := instance2.GetMachineID()

		if id1 != id2 {
			t.Errorf("Machine IDs are not consistent. First: %s, Second: %s", id1, id2)
		} else {
			t.Logf("Machine ID is consistent: %s", id1)
		}
	})
}
