package core

import (
	"context"
	"io"
	"log"

	"github.com/dush-t/ryz/core/entities"
	"google.golang.org/grpc"

	p4ConfigV1 "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4V1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// Client contains all the information required to handle a client
type Client struct {
	p4V1.P4RuntimeClient
	deviceID               uint64
	isMaster               bool
	electionID             p4V1.Uint128
	p4Info                 *p4ConfigV1.P4Info
	IncomingMessageChannel chan *p4V1.StreamMessageResponse
	OutgoingMessageChannel chan *p4V1.StreamMessageRequest
	streamChannel          p4V1.P4Runtime_StreamChannelClient
	Entities               map[entities.EntityType]*(map[string]entities.Entity)
}

// Init will create a new gRPC connection and initialize the client
func (c *Client) Init(addr string, deviceID uint64, electionID p4V1.Uint128) error {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return err
	}

	p4RtC := p4V1.NewP4RuntimeClient(conn)
	resp, err := p4RtC.Capabilities(context.Background(), &p4V1.CapabilitiesRequest{})
	if err != nil {
		log.Fatal("Error in capabilities RPC", err)
	}
	log.Println("P4Runtime server version is", resp.P4RuntimeApiVersion)

	streamMsgs := make(chan *p4V1.StreamMessageResponse, 20)
	pushMsgs := make(chan *p4V1.StreamMessageRequest)

	c.P4RuntimeClient = p4RtC
	c.deviceID = deviceID
	c.electionID = electionID
	c.IncomingMessageChannel = streamMsgs
	c.OutgoingMessageChannel = pushMsgs

	stream, streamInitErr := c.StreamChannel(context.Background())
	if streamInitErr != nil {
		return streamInitErr
	}

	c.streamChannel = stream

	return nil
}

// Run will do whatever is needed to ensure that the client is active
// once it is initialized.
func (c *Client) Run() {
	c.StartMessageChannels()
}

// WriteUpdate is used to update an entity on the
// switch. Refer to the P4Runtime spec to know more.
func (c *Client) WriteUpdate(update *p4V1.Update) error {
	req := &p4V1.WriteRequest{
		DeviceId:   c.deviceID,
		ElectionId: &c.electionID,
		Updates:    []*p4V1.Update{update},
	}

	_, err := c.Write(context.Background(), req)
	return err
}

// ReadEntities will return a channel on which it will keep on sending all
// the entities that are returned from our request. If you don't want to
// receive these entries sequentially (through a channel) and want them all
// at once, just use ReadEntitiesSync
func (c *Client) ReadEntities(entities []*p4V1.Entity) (chan *p4V1.Entity, error) {
	req := &p4V1.ReadRequest{
		DeviceId: c.deviceID,
		Entities: entities,
	}
	stream, err := c.Read(context.TODO(), req)
	if err != nil {
		return nil, err
	}

	entityChannel := make(chan *p4V1.Entity)
	go func() {
		defer close(entityChannel)
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			for _, e := range res.Entities {
				entityChannel <- e
			}
		}
	}()

	return entityChannel, nil
}

// ReadEntitiesSync will call ReadEntities, accumulate the results and return them all
// at once. Nothing fancy here.
func (c *Client) ReadEntitiesSync(entities []*p4V1.Entity) ([]*p4V1.Entity, error) {
	entityChannel, err := c.ReadEntities(entities)
	if err != nil {
		return nil, err
	}

	result := make([]*p4V1.Entity, 1)
	for e := range entityChannel {
		result = append(result, e)
	}

	return result, nil
}

// NewClient will create a new P4 Runtime Client
func NewClient(addr string, deviceID uint64, electionID p4V1.Uint128) (P4RClient, error) {
	client := &Client{}
	initErr := client.Init(addr, deviceID, electionID)
	if initErr != nil {
		return nil, initErr
	}

	return client, nil
}

/*
	Getters and Setters beyond this point
*/

// GetMessageChannels will return the message channels used by the client
func (c *Client) GetMessageChannels() MessageChannels {
	return MessageChannels{
		IncomingMessageChannel: c.IncomingMessageChannel,
		OutgoingMessageChannel: c.OutgoingMessageChannel,
	}
}

// GetArbitrationData will return the data required to perform arbitration
// for the client
func (c *Client) GetArbitrationData() ArbitrationData {
	return ArbitrationData{
		DeviceID:   c.deviceID,
		ElectionID: c.electionID,
	}
}

// GetStreamChannel will return the StreamChannel instance associated with the client
func (c *Client) GetStreamChannel() p4V1.P4Runtime_StreamChannelClient {
	return c.streamChannel
}

// P4Info will return the P4Info struct associated to the client
func (c *Client) P4Info() *p4ConfigV1.P4Info {
	return c.p4Info
}

// IsMaster returns true if the client is master
func (c *Client) IsMaster() bool {
	return c.isMaster
}

// SetMastershipStatus sets the mastership status of the client
func (c *Client) SetMastershipStatus(status bool) {
	c.isMaster = status
}

// GetEntities will return the Entities that the client has
func (c *Client) GetEntities(EntityType entities.EntityType) *map[string]entities.Entity {
	return c.Entities[EntityType]
}
