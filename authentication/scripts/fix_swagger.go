package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	swaggerFile := "docs/swagger/realio-authentication.swagger.json"
	data, err := os.ReadFile(swaggerFile)
	if err != nil {
		fmt.Printf("Error reading swagger file: %v\n", err)
		os.Exit(1)
	}

	var swagger map[string]interface{}
	if err := json.Unmarshal(data, &swagger); err != nil {
		fmt.Printf("Error parsing swagger JSON: %v\n", err)
		os.Exit(1)
	}

	paths, ok := swagger["paths"].(map[string]interface{})
	if !ok {
		fmt.Println("Could not find paths in swagger")
		os.Exit(1)
	}

	uploadPath, ok := paths["/api/v1/upload-image"].(map[string]interface{})
	if !ok {
		fmt.Println("Could not find /api/v1/upload-image in paths")
		os.Exit(1)
	}

	post, ok := uploadPath["post"].(map[string]interface{})
	if !ok {
		fmt.Println("Could not find POST method for /api/v1/upload-image")
		os.Exit(1)
	}

	post["parameters"] = []map[string]interface{}{
		{
			"name":        "userId",
			"in":          "formData",
			"description": "User ID",
			"required":    true,
			"type":        "string",
		},
		{
			"name":        "content",
			"in":          "formData",
			"description": "Image file to upload",
			"required":    true,
			"type":        "file",
		},
	}

	post["consumes"] = []string{"multipart/form-data"}

	modifiedData, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling modified swagger: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(swaggerFile, modifiedData, 0644); err != nil {
		fmt.Printf("Error writing modified swagger: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully updated swagger.json with file upload parameters")
}
