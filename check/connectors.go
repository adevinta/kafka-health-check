package check

import (
	"fmt"
	"time"

	"github.com/Adevinta/kafka/v2"
	"github.com/Adevinta/kafka/v2/proto"
	"github.com/samuel/go-zookeeper/zk"
)

// BrokerConnection represents a connection to the Kafka broker
type BrokerConnection interface {
	Dial(nodeAddresses []string, conf kafka.BrokerConf) error

	Consumer(conf kafka.ConsumerConf) (kafka.Consumer, error)

	Producer(conf kafka.ProducerConf) kafka.Producer

	Metadata() (*proto.MetadataResp, error)

	CreateTopic(proto.TopicInfo, time.Duration) error

	DeleteTopic(string, time.Duration) error

	Close()
}

// actual implementation of the Kafka broker connection based on optiopay/kafka.
type kafkaBrokerConnection struct {
	broker *kafka.Broker
}

func (connection *kafkaBrokerConnection) Dial(nodeAddresses []string, conf kafka.BrokerConf) error {
	broker, err := kafka.Dial(nodeAddresses, conf)
	if err != nil {
		return err
	}
	connection.broker = broker
	return nil
}

func (connection *kafkaBrokerConnection) Consumer(conf kafka.ConsumerConf) (kafka.Consumer, error) {
	return connection.broker.Consumer(conf)
}

func (connection *kafkaBrokerConnection) CreateTopic(topic proto.TopicInfo, timeout time.Duration) error {
	resp, err := connection.broker.CreateTopic([]proto.TopicInfo{topic}, timeout, false)
	if err != nil {
		return err
	}

	if len(resp.TopicErrors) > 0 {
		if resp.TopicErrors[0].ErrorCode != 0 {
			return resp.TopicErrors[0].Err
		}
	}

	return nil
}

func (connection *kafkaBrokerConnection) DeleteTopic(topic string, timeout time.Duration) error {
	resp, err := connection.broker.DeleteTopic([]string{topic}, timeout)
	if err != nil {
		return err
	}

	if len(resp.TopicErrors) > 0 {
		if resp.TopicErrors[0].ErrorCode != 0 {
			return fmt.Errorf("Deletion failed for Topic: %s with ErrorCode: %d", resp.TopicErrors[0].Topic, resp.TopicErrors[0].ErrorCode)
		}
	}

	return nil
}

func (connection *kafkaBrokerConnection) Producer(conf kafka.ProducerConf) kafka.Producer {
	return connection.broker.Producer(conf)
}

func (connection *kafkaBrokerConnection) Metadata() (*proto.MetadataResp, error) {
	return connection.broker.Metadata()
}

func (connection *kafkaBrokerConnection) Close() {
	connection.broker.Close()
}

// ZkConnection represents a connection to a ZooKeeper ensemble
type ZkConnection interface {
	Connect(servers []string, sessionTimeout time.Duration) (<-chan zk.Event, error)
	Close()
	Exists(path string) (bool, *zk.Stat, error)
	Set(path string, data []byte, version int32) (*zk.Stat, error)
	Create(path string, data []byte, flags int32, acl []zk.ACL) (string, error)
	Children(path string) ([]string, *zk.Stat, error)
	Get(path string) ([]byte, *zk.Stat, error)
	Lock(path string) error
	Unlock(path string) error
}

// Actual implementation based on samuel/go-zookeeper/zk
type zkConnection struct {
	connection *zk.Conn
	locks      map[string]*zk.Lock
}

type zkNullLogger struct {
}

func (zkNullLogger) Printf(string, ...interface{}) {}

func (zkConn *zkConnection) Connect(servers []string, sessionTimeout time.Duration) (<-chan zk.Event, error) {
	loggerOption := func(c *zk.Conn) {
		c.SetLogger(zkNullLogger{})
	}
	connection, events, err := zk.Connect(servers, sessionTimeout, loggerOption)
	zkConn.connection = connection
	zkConn.locks = map[string]*zk.Lock{}
	return events, err
}

func (zkConn *zkConnection) Close() {
	zkConn.connection.Close()
	zkConn.locks = nil
}

func (zkConn *zkConnection) Exists(path string) (bool, *zk.Stat, error) {
	return zkConn.connection.Exists(path)
}

func (zkConn *zkConnection) Set(path string, data []byte, version int32) (*zk.Stat, error) {
	return zkConn.connection.Set(path, data, version)
}

func (zkConn *zkConnection) Create(path string, data []byte, flags int32, acl []zk.ACL) (string, error) {
	return zkConn.connection.Create(path, data, flags, acl)
}

func (zkConn *zkConnection) Children(path string) ([]string, *zk.Stat, error) {
	return zkConn.connection.Children(path)
}

func (zkConn *zkConnection) Get(path string) ([]byte, *zk.Stat, error) {
	return zkConn.connection.Get(path)
}

func (zkConn *zkConnection) Lock(path string) error {
	if zkConn.locks == nil {
		return fmt.Errorf("connection not initialized")
	}
	if _, ok := zkConn.locks[path]; !ok {
		zkConn.locks[path] = zk.NewLock(zkConn.connection, path, zk.WorldACL(zk.PermAll))
	}
	lock := zkConn.locks[path]

	return lock.Lock()
}

func (zkConn *zkConnection) Unlock(path string) error {
	if zkConn.locks == nil {
		return fmt.Errorf("connection not initialized")
	}
	if lock, ok := zkConn.locks[path]; !ok {
		return fmt.Errorf("not locked")
	} else {
		return lock.Unlock()
	}
}
