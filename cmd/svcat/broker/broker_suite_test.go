package broker_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	_ "github.com/kubernetes-incubator/service-catalog/internal/test"
)

func TestBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Broker Suite")
}
