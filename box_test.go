package main

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/suite"
	"github.com/wercker/sentcli/util"
)

func boxByID(s string) (*Box, error) {
	return NewBox(&BoxConfig{ID: s}, emptyPipelineOptions(), &BoxOptions{})
}

type BoxSuite struct {
	*util.TestSuite
}

func TestBoxSuite(t *testing.T) {
	suiteTester := &BoxSuite{&util.TestSuite{}}
	suite.Run(t, suiteTester)
}

func (s *BoxSuite) TestName() {
	_, err := boxByID("wercker/base@1.0.0")
	s.NotNil(err)

	noTag, err := boxByID("wercker/base")
	s.Nil(err)
	s.Equal("wercker/base:latest", noTag.Name)

	withTag, err := boxByID("wercker/base:foo")
	s.Nil(err)
	s.Equal("wercker/base:foo", withTag.Name)
}

func (s *BoxSuite) TestPortBindings() {
	published := []string{
		"8000",
		"8001:8001",
		"127.0.0.1::8002",
		"127.0.0.1:8004:8003/udp",
	}
	checkBindings := [][]string{
		[]string{"8000/tcp", "", "8000"},
		[]string{"8001/tcp", "", "8001"},
		[]string{"8002/tcp", "127.0.0.1", "8002"},
		[]string{"8003/udp", "127.0.0.1", "8004"},
	}

	bindings := portBindings(published)
	s.Equal(len(checkBindings), len(bindings))
	for _, check := range checkBindings {
		binding := bindings[docker.Port(check[0])]
		s.Equal(check[1], binding[0].HostIP)
		s.Equal(check[2], binding[0].HostPort)
	}
}
