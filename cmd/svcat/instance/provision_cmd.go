/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"fmt"

	"github.com/kubernetes-incubator/service-catalog/cmd/svcat/command"
	"github.com/kubernetes-incubator/service-catalog/cmd/svcat/output"
	"github.com/kubernetes-incubator/service-catalog/cmd/svcat/parameters"
	servicecatalog "github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"
	"github.com/spf13/cobra"
)

type ProvisionCmd struct {
	*command.Namespaced
	*command.Waitable

	InstanceName string
	ExternalID   string
	ClassName    string
	PlanName     string
	RawParams    []string
	JsonParams   string
	Params       interface{}
	RawSecrets   []string
	Secrets      map[string]string
}

// NewProvisionCmd builds a "svcat provision" command
func NewProvisionCmd(cxt *command.Context) *cobra.Command {
	provisionCmd := &ProvisionCmd{
		Namespaced: command.NewNamespaced(cxt),
		Waitable:   command.NewWaitable(),
	}
	cmd := &cobra.Command{
		Use:   "provision NAME --plan PLAN --class CLASS",
		Short: "Create a new instance of a service",
		Example: command.NormalizeExamples(`
  svcat provision wordpress-mysql-instance --class mysqldb --plan free -p location=eastus -p sslEnforcement=disabled
  svcat provision wordpress-mysql-instance --external-id a7c00676-4398-11e8-842f-0ed5f89f718b --class mysqldb --plan free
  svcat provision wordpress-mysql-instance --class mysqldb --plan free -s mysecret[dbparams]
  svcat provision secure-instance --class mysqldb --plan secureDB --params-json '{
    "encrypt" : true,
    "firewallRules" : [
        {
            "name": "AllowSome",
            "startIPAddress": "75.70.113.50",
            "endIPAddress" : "75.70.113.131"
        }
    ]
  }'
`),
		PreRunE: command.PreRunE(provisionCmd),
		RunE:    command.RunE(provisionCmd),
	}
	cmd.Flags().StringVar(&provisionCmd.PlanName, "plan", "",
		"The plan name (Required)")
	cmd.MarkFlagRequired("plan")
	cmd.Flags().StringVar(&provisionCmd.ClassName, "class", "",
		"The class name (Required)")
	cmd.MarkFlagRequired("class")
	cmd.Flags().StringVar(&provisionCmd.ExternalID, "external-id", "",
		"The ID of the instance for use with the OSB SB API (Optional)")
	cmd.Flags().StringSliceVarP(&provisionCmd.RawParams, "param", "p", nil,
		"Additional parameter to use when provisioning the service, format: NAME=VALUE. Cannot be combined with --params-json, Sensitive information should be placed in a secret and specified with --secret")
	cmd.Flags().StringSliceVarP(&provisionCmd.RawSecrets, "secret", "s", nil,
		"Additional parameter, whose value is stored in a secret, to use when provisioning the service, format: SECRET[KEY]")
	cmd.Flags().StringVar(&provisionCmd.JsonParams, "params-json", "",
		"Additional parameters to use when provisioning the service, provided as a JSON object. Cannot be combined with --param")
	provisionCmd.AddNamespaceFlags(cmd.Flags(), false)
	provisionCmd.AddWaitFlags(cmd)

	return cmd
}

func (c *ProvisionCmd) Validate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("an instance name is required")
	}
	c.InstanceName = args[0]

	var err error

	if c.JsonParams != "" && len(c.RawParams) > 0 {
		return fmt.Errorf("--params-json cannot be used with --param")
	}

	if c.JsonParams != "" {
		c.Params, err = parameters.ParseVariableJSON(c.JsonParams)
		if err != nil {
			return fmt.Errorf("invalid --params-json value (%s)", err)
		}
	} else {
		c.Params, err = parameters.ParseVariableAssignments(c.RawParams)
		if err != nil {
			return fmt.Errorf("invalid --param value (%s)", err)
		}
	}

	c.Secrets, err = parameters.ParseKeyMaps(c.RawSecrets)
	if err != nil {
		return fmt.Errorf("invalid --secret value (%s)", err)
	}

	return nil
}

func (c *ProvisionCmd) Run() error {
	return c.Provision()
}

func (c *ProvisionCmd) Provision() error {
	opts := &servicecatalog.ProvisionOptions{
		ExternalID: c.ExternalID,
		Namespace:  c.Namespace,
		Params:     c.Params,
		Secrets:    c.Secrets,
	}
	instance, err := c.App.Provision(c.InstanceName, c.ClassName, c.PlanName, opts)
	if err != nil {
		return err
	}

	if c.Wait {
		fmt.Fprintln(c.Output, "Waiting for the instance to be provisioned...")
		finalInstance, err := c.App.WaitForInstance(instance.Namespace, instance.Name, c.Interval, c.Timeout)
		if err == nil {
			instance = finalInstance
		}

		// Always print the instance because the provision did succeed,
		// and just print any errors that occurred while polling
		output.WriteInstanceDetails(c.Output, instance)
		return err
	}

	output.WriteInstanceDetails(c.Output, instance)
	return nil
}
