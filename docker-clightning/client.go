package dockerCLightning

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lncm/invoicer/common"
	"github.com/pkg/errors"
	"os/exec"
)

const ClientName = "docker-clightning"

type (
	DockerCLightning struct {
		Network string
	}

	// NOTE: all non-critical fields temporarily commented out

	// Network Info
	networkInfo struct {
		//Type    string `json:"type"`
		Address string `json:"address"`
		Port    int    `json:"port"`
	}

	Info struct {
		Id string `json:"id"`
		//Alias       string        `json:"alias"`
		//Version     string        `json:"version"`
		//Network     string        `json:"network"`
		//BlockHeight int           `json:"blockheight"`
		Address []networkInfo `json:"address"`
		//Binding     []networkInfo `json:"binding"`
	}
)

func getRawInfo(c *gin.Context) {
	info, err := exec.Command("/usr/bin/docker", "exec", "lightningpay", "lightning-cli", "getinfo").Output()
	if err == nil {
		c.String(200, fmt.Sprintf("%s", info))
		return
	}

	c.JSON(500, gin.H{
		"error": fmt.Sprintf("Error from lightning service: %s", err),
	})
	return

}

func (dockerCLightning DockerCLightning) Invoice(amount float64, desc string) (invoice common.Invoice, err error) {
	return invoice, errors.New("not implemented yet")
}

func (dockerCLightning DockerCLightning) Status(hash string) (s common.Status, err error) {
	return s, errors.New("not implemented yet")
}

func (dockerCLightning DockerCLightning) Info() (info common.Info, err error) {
	out, err := exec.Command("/usr/bin/docker", "exec", "lightningpay", "lightning-cli", "getinfo").Output()
	if err != nil {
		return
	}

	var rawInfo Info
	err = json.Unmarshal(out, &rawInfo)
	if err != nil {
		return info, errors.Wrap(err, "unable to decode response")
	}

	if len(rawInfo.Address) == 0 {
		return info, errors.New("unable to get any connstrings")
	}

	for _, address := range rawInfo.Address {
		info.Uris = append(info.Uris, fmt.Sprintf("%s@%s:%d", rawInfo.Id, address.Address, address.Port))
	}

	return info, nil
}

func New(network string) DockerCLightning {
	return DockerCLightning{network}
}
