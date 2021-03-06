package main

import (
	"log"

	"github.com/dush-t/ryz/core/entities"
	"github.com/dush-t/ryz/util"
)

func ipv4LpmTransform(data map[string]interface{}) ([]entities.Match, [][]byte) {
	byteIP, err := util.IpToBinary(data["ip"].(string))
	bytePort, err := util.UInt32ToBinary(data["port"].(uint32), 2)
	byteMac, err := util.MacToBinary(data["mac"].(string))
	if err != nil {
		log.Println("Invalid inputs in ipv4LPM table. You messed up.", err)
	}

	matchFields := []entities.Match{&entities.LpmMatch{
		Value: byteIP,
		PLen:  32,
	}}
	params := []([]byte){byteMac, bytePort}

	return matchFields, params
}
