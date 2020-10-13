package control

import (
	"log"

	"github.com/dush-t/ryz/core"
	"github.com/dush-t/ryz/core/entities"

	p4V1 "github.com/p4lang/p4runtime/go/p4/v1"
)

// SimpleControl implements the Control interface.
// I could not think of a better name
type SimpleControl struct {
	Client             core.P4RClient
	DigestChannel      chan *p4V1.StreamMessageResponse_Digest
	ArbitrationChannel chan *p4V1.StreamMessageResponse_Arbitration
}

// StartMessageRouter will start a goroutine that takes incoming messages from the stream
// and then sends them to corresponding channels based on message types
func (sc *SimpleControl) StartMessageRouter() {
	IncomingMessageChannel := sc.Client.GetMessageChannels().IncomingMessageChannel
	go func() {
		for {
			in := <-IncomingMessageChannel
			update := in.GetUpdate()

			switch update.(type) {
			case *p4V1.StreamMessageResponse_Arbitration:
				sc.ArbitrationChannel <- update.(*p4V1.StreamMessageResponse_Arbitration)
			case *p4V1.StreamMessageResponse_Digest:
				sc.DigestChannel <- update.(*p4V1.StreamMessageResponse_Digest)
			default:
				log.Println("Message has unknown type")
			}
		}
	}()
}

// Table will return a TableControl struct
func (sc *SimpleControl) Table(tableName string) TableControl {
	tables := *(sc.Client.GetEntities(entities.EntityTypes.TABLE))
	table := tables[tableName].(*entities.Table)

	return TableControl{
		table:   table,
		control: sc,
	}
}

// SetMastershipStatus will call a method of the same name on P4RClient.
// We need to keep track of mastership to reason about which control can be
// used for what.
func (sc *SimpleControl) SetMastershipStatus(status bool) {
	sc.Client.SetMastershipStatus(status)
}

// IsMaster will return true if the control has mastership
func (sc *SimpleControl) IsMaster() bool {
	return sc.Client.IsMaster()
}

// Run will do all the work required to actually get the control
// instance up and running
func (sc *SimpleControl) Run() {
	// Start running the client i.e. start the Stream Channel
	// on the client.
	sc.Client.Run()

	// Start the goroutine that will take messages from the
	// streamchannel and route it to appropriate goroutines for
	// handling.
	sc.StartMessageRouter()

	// Start the goroutine that listens to arbitration updates
	// and handles those updates.
	sc.StartArbitrationUpdateListener()

	// Perform arbitration
	sc.PerformArbitration()
}

// NewControl will create a new Control instance
func NewControl(addr, p4InfoPath string, deviceID uint64, electionID p4V1.Uint128) (Control, error) {
	client, err := core.NewClient(addr, p4InfoPath, deviceID, electionID)
	if err != nil {
		return nil, err
	}
	digestChan := make(chan *p4V1.StreamMessageResponse_Digest, 10)
	arbitrationChan := make(chan *p4V1.StreamMessageResponse_Arbitration)

	control := SimpleControl{
		Client:             client,
		DigestChannel:      digestChan,
		ArbitrationChannel: arbitrationChan,
	}

	return &control, nil
}
