package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"
)

func generateInventory(config Config) error {
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = "src/output/terraform"
	out, err := cmd.Output()
	if err != nil {
		// Check if the error is due to terraform output not being available
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "No outputs found") || strings.Contains(string(exitErr.Stderr), "no such file or directory") {
				return fmt.Errorf("terraform output not available. Please run 'terraform -chdir=src/output/terraform apply' first.")
			}
		}
		return fmt.Errorf("error running terraform output: %w", err)
	}

	// New check for empty output
	if len(out) == 0 || strings.TrimSpace(string(out)) == "{}" {
		return fmt.Errorf("terraform output is empty. Please run 'terraform -chdir=src/output/terraform apply' first to create resources.")
	}

	var tfOutput TerraformOutput
	if err := json.Unmarshal(out, &tfOutput); err != nil {
		return fmt.Errorf("error unmarshalling terraform output: %w", err)
	}

	k3sCluster := K3sCluster{
		Vars: config.K3s.Vars,
	}

	for _, ip := range tfOutput.K3sMasterPrivateIP.Value {
		k3sCluster.Master = append(k3sCluster.Master, Node{Ip: ip})
	}

	for _, ip := range tfOutput.K3sWorkerPrivateIPs.Value {
		k3sCluster.Worker = append(k3sCluster.Worker, Node{Ip: ip})
	}

	tmplPath := "src/tmpls/yml/k3s_inventory.yml.tmpl"

	outputDir := filepath.Dir("src/output/yml/k3s_inventory.yml")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("error parsing inventory template: %w", err)
	}

	outputFile, err := os.Create("src/output/yml/k3s_inventory.yml")
	if err != nil {
		return fmt.Errorf("error creating inventory file: %w", err)
	}
	defer outputFile.Close()

	data := map[string]interface{}{
		"k3s_cluster": k3sCluster,
	}

	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("error executing inventory template: %w", err)
	}

	return nil
}

func generateKubectlScript(config Config) error {
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Dir = "src/output/terraform"
	out, err := cmd.Output()
	if err != nil {
		// Check if the error is due to terraform output not being available
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "No outputs found") || strings.Contains(string(exitErr.Stderr), "no such file or directory") {
				return fmt.Errorf("terraform output not available. Please run 'terraform -chdir=src/output/terraform apply' first.")
			}
		}
		return fmt.Errorf("error running terraform output: %w", err)
	}

	// New check for empty output
	if len(out) == 0 || strings.TrimSpace(string(out)) == "{}" {
		return fmt.Errorf("terraform output is empty. Please run 'terraform -chdir=src/output/terraform apply' first to create resources.")
	}

	var tfOutput TerraformOutput
	if err := json.Unmarshal(out, &tfOutput); err != nil {
		return fmt.Errorf("error unmarshalling terraform output: %w", err)
	}

	k3sCluster := K3sCluster{
		Vars: config.K3s.Vars,
	}

	k3sCluster.Master = append(k3sCluster.Master, Node{Ip: tfOutput.MasterFloatingIP.Value})

	tmplPath := "src/tmpls/script/setup_network.sh.tmpl"

	outputDir := filepath.Dir("src/output/script/setup_network.sh")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("error parsing kubectl script template: %w", err)
	}

	outputFile, err := os.Create("src/output/script/setup_network.sh")
	if err != nil {
		return fmt.Errorf("error creating kubectl script file: %w", err)
	}
	defer outputFile.Close()

	data := map[string]interface{}{
		"k3s_cluster": k3sCluster,
	}

	if err := tmpl.Execute(outputFile, data); err != nil {
		return fmt.Errorf("error executing kubectl script template: %w", err)
	}

	return nil
}

func executeTemplate(templatePath string, config Config) error {
	relPath, err := filepath.Rel("src/tmpls", templatePath)
	if err != nil {
		return err
	}

	outputDir := filepath.Join("src/output", filepath.Dir(relPath))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"ssh_public_key_path": func() string {
			return config.SshPublicKeyPath
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
		"loop": func(n int) []int {
			var s []int
			for i := 0; i < n; i++ {
				s = append(s, i)
			}
			return s
		},
	}).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("error parsing template file %s: %w", templatePath, err)
	}

	outputFile, err := os.Create(filepath.Join("src/output", strings.TrimSuffix(relPath, ".tmpl")))
	if err != nil {
		return fmt.Errorf("error creating output file for %s: %w", templatePath, err)
	}
	defer outputFile.Close()

	if err := tmpl.Execute(outputFile, config); err != nil {
		return fmt.Errorf("error executing template for %s: %w", templatePath, err)
	}

	return nil
}

func main() {
	var config Config

	files := []string{"input.yml", "input.local.yml"}

	for _, file := range files {
		yamlFile, err := ioutil.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Printf("Error reading YAML file %s: %s\n", file, err)
			return
		}

		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			fmt.Printf("Error unmarshalling YAML file %s: %s\n", file, err)
			return
		}
	}

	// Find all template files in the tmpls directory
	walkErr := filepath.Walk("src/tmpls", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip the inventory template, as it is handled separately
		if path == "src/tmpls/yml/k3s_inventory.yml.tmpl" || path == "src/tmpls/script/setup_network.sh.tmpl" {
			return nil
		}

		return executeTemplate(path, config)
	})

	if walkErr != nil {
		fmt.Printf("Error processing templates: %s\n", walkErr)
		return
	}

	fmt.Println("Successfully generated files from templates.")

	if err := generateInventory(config); err != nil {
		if strings.Contains(err.Error(), "terraform output not available") || strings.Contains(err.Error(), "terraform output is empty") {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Error generating inventory: %s\n", err)
		}
		return
	}

	fmt.Println("Successfully generated inventory.")

	if err := generateKubectlScript(config); err != nil {
		if strings.Contains(err.Error(), "terraform output not available") || strings.Contains(err.Error(), "terraform output is empty") {
			fmt.Println(err.Error())
		} else {
			fmt.Printf("Error generating kubectl script: %s\n", err)
		}
		return
	}

	fmt.Println("Successfully generated network setup script.")
}
