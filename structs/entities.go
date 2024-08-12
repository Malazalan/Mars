package structs

import (
	"Mars/icarus"
	"fmt"
)

type EntityInit struct {
	Class    string
	Lat      float32
	Lng      float32
	Alt      float32
	Id       uint64
	Name     string
	Nation   string
	Health   uint32
	Payloads map[string]any
}

func NewEntity(class string, lat float32, lng float32, alt float32, id uint64, name string, nation string,
	health uint32, payloads map[string]any) EntityInit {
	return EntityInit{
		Class:    class,
		Lat:      lat,
		Lng:      lng,
		Alt:      alt,
		Id:       id,
		Name:     name,
		Nation:   nation,
		Health:   health,
		Payloads: payloads,
	}
}

func UpdateEntity(entity EntityInit, attribute string, value any) EntityInit {
	switch attribute {
	case "class":
		entity.Class = value.(string)
		break
	case "lat":
		entity.Lat = value.(float32)
		break
	case "lng":
		entity.Lng = value.(float32)
		break
	case "alt":
		entity.Alt = value.(float32)
	case "id":
		entity.Id = value.(uint64)
		break
	case "name":
		entity.Name = value.(string)
		break
	case "nation":
		entity.Nation = value.(string)
		break
	case "health":
		entity.Health = value.(uint32)
		break
	case "payloads":
		payloadToChange := entity.Payloads[fmt.Sprintf("%d", value.([]any)[0])].(*icarus.PayloadStatus)
		payloadToChange.CurrentQuantity = value.([]any)[1].(uint64)
		break
	}
	return entity
}
