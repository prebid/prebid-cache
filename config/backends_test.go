package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAerospikeValidateAndLog(t *testing.T) {
	testCases := []struct {
		desc          string
		inCfg         Aerospike
		hasError      bool
		expectedError error
	}{
		{
			desc: "aerospike.hosts passed in",
			inCfg: Aerospike{
				Hosts: []string{"foo.com", "bat.com"},
				Port:  8888,
			},
			hasError: false,
		},
		{
			desc: "aerospike.host passed in",
			inCfg: Aerospike{
				Host: "foo.com",
				Port: 8888,
			},
			hasError: false,
		},
		{
			desc: "aerospike.host aerospike.hosts passed in",
			inCfg: Aerospike{
				Host:  "foo.com",
				Hosts: []string{"foo.com", "bat.com"},
				Port:  8888,
			},
			hasError: false,
		},
		{
			desc: "aerospike.host and aerospike.hosts missing",
			inCfg: Aerospike{
				Port: 8888,
			},
			hasError:      true,
			expectedError: fmt.Errorf("Cannot connect to empty Aerospike host(s)"),
		},
		{
			desc: "aerospike.port config missing",
			inCfg: Aerospike{
				Host: "foo.com",
			},
			hasError:      true,
			expectedError: fmt.Errorf("Cannot connect to Aerospike host at port 0"),
		},
	}

	for _, test := range testCases {

		//run test
		if test.hasError {
			assert.Equal(t, test.inCfg.validateAndLog(), test.expectedError, test.desc)
		} else {
			assert.Nil(t, test.inCfg.validateAndLog(), test.desc)
		}
	}
}
