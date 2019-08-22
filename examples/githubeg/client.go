package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/perlin-network/noise"
	"github.com/perlin-network/noise/cipher/aead"
	"github.com/perlin-network/noise/handshake/ecdh"
	"github.com/perlin-network/noise/log"
	"github.com/perlin-network/noise/payload"
	"github.com/perlin-network/noise/protocol"
	"github.com/perlin-network/noise/skademlia"
	"github.com/pkg/errors"
)

/** DEFINE MESSAGES **/
var (
	opcodeChat noise.Opcode
	_          noise.Message = (*chatMessage)(nil)
)

type chatMessage struct {
	text string
}

func myFunc() {
	fmt.Println("MY FUNCTION CALLED")
}
func (chatMessage) Read(reader payload.Reader) (noise.Message, error) {
	text, err := reader.ReadString()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read chat msg")
	}
	fmt.Println("IN THE READ FUNCTION")

	return chatMessage{text: text}, nil
}

func (m chatMessage) Write() []byte {

	fmt.Println("IN THE WRITEEEE FUNCTION")
	return payload.NewWriter(nil).WriteString(m.text).Bytes()
}

/** ENTRY POINT **/
func setup(node *noise.Node) {
	opcodeChat = noise.RegisterMessage(noise.NextAvailableOpcode(), (*chatMessage)(nil))

	node.OnPeerInit(func(node *noise.Node, peer *noise.Peer) error {
		peer.OnConnError(func(node *noise.Node, peer *noise.Peer, err error) error {
			log.Info().Msgf("Got an error: %v", err)

			return nil
		})

		peer.OnDisconnect(func(node *noise.Node, peer *noise.Peer) error {
			log.Info().Msgf("Peer %v has disconnected.", peer.RemoteIP().String()+":"+strconv.Itoa(int(peer.RemotePort())))

			return nil
		})

		go func() {
			for {
				msg := <-peer.Receive(opcodeChat)
				//	log.Info().Msgf("[%s]: %s", protocol.PeerID(peer), msg.(chatMessage).text)
				if msg.(chatMessage).text == "start" {
					myFunc()
					s := []byte("start")
					payload := make([]byte, 1024*1024*7)
					_, _ = rand.Read(payload)
					payload = append(s, payload...)

					fmt.Println("Broadcast Shuru")
					skademlia.Broadcast(node, chatMessage{text: string(payload)})
					fmt.Println("Broadcast khattam")
				}

			}
		}()

		return nil
	})
}

func main() {
	hostFlag := flag.String("h", "127.0.0.1", "host to listen for peers on")
	portFlag := flag.Uint("p", 3000, "port to listen for peers on")
	flag.Parse()

	params := noise.DefaultParams()
	//params.NAT = nat.NewPMP()
	params.Keys = skademlia.RandomKeys()
	params.Host = *hostFlag
	params.Port = uint16(*portFlag)

	fmt.Println("MAXIMUM MESSAGE SIZE IS")
	fmt.Println(params.MaxMessageSize)

	node, err := noise.NewNode(params)
	if err != nil {
		panic(err)
	}

	defer node.Kill()

	p := protocol.New()
	p.Register(ecdh.New())
	p.Register(aead.New())
	p.Register(skademlia.New())
	p.Enforce(node)

	setup(node)
	go node.Listen()

	log.Info().Msgf("Listening for peers on port %d.", node.ExternalPort())

	if len(flag.Args()) > 0 {
		for _, address := range flag.Args() {
			peer, err := node.Dial(address)
			fmt.Print("dialing peer --")
			fmt.Println(peer)

			if err != nil {
				panic(err)
			}

			skademlia.WaitUntilAuthenticated(peer)
		}

		peers := skademlia.FindNode(node, protocol.NodeID(node).(skademlia.ID), skademlia.BucketSize(), 8)
		log.Info().Msgf("Bootstrapped with peers: %+v", peers)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		txt, err := reader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		skademlia.BroadcastAsync(node, chatMessage{text: strings.TrimSpace(txt)})
	}
}
