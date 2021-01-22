package remote

import (
	"io"

	"github.com/shipyard-run/connector/protos/shipyard"
)

func (s *Server) handleConnectionRead(serviceID string, si *streamInfo, svc *service, conn *bufferedConn) {
	s.log.Info(
		"listener",
		"message", "Read from connection for",
		"service_id", serviceID)

	// message id allows the reordering of messages in case they arrive out of sequence
	var messageID int32 = 0

	// read the data from the connection
	for {
		data := make([]byte, MessageSize)

		s.log.Debug("listener", "message", "Reading data from connection", "service_id", serviceID, "connection_id", conn.id)

		// read 4K of data from the connection
		i, err := conn.Read(data)

		// unable to read the data, kill the connection
		if err != nil || i == 0 {
			if err == io.EOF {
				s.log.Debug(
					"listener",
					"message", "Connection closed",
					"service_id", serviceID,
					"connection_id", conn.id,
					"error", err)

			} else {
				s.log.Error(
					"listener",
					"message", "Unable to read data from the connection",
					"service_id", serviceID,
					"connection_id", conn.id,
					"error", err)
			}

			// the connection has closed
			// notify the remote
			si.grpcConn.Send(
				&shipyard.OpenData{
					ServiceId:    serviceID,
					ConnectionId: conn.id,
					Message:      &shipyard.OpenData_Closed{Closed: &shipyard.Closed{}},
				},
			)

			// exit the for loop
			return
		}

		s.log.Trace(
			"listener",
			"message", "Read data from connection",
			"service_id", serviceID,
			"connection_id", conn.id,
			"len", i,
			"data", string(data[:i]))

		// send the read chunk of data over the gRPC stream
		// check there is a remote connection if not just return
		s.log.Debug(
			"listener",
			"message", "Sending data to remote server",
			"id", messageID,
			"addr", si.addr,
			"service_id", serviceID,
			"connection_id", conn.id)

		si.grpcConn.Send(
			&shipyard.OpenData{
				ServiceId:    serviceID,
				ConnectionId: conn.id,
				Message:      &shipyard.OpenData_Data{Data: &shipyard.Data{Id: messageID, Data: data[:i]}},
			},
		)

		// increment the messageid
		messageID++

		// we have read all the data send the other end a message so it knows it can now send a response
		if i < MessageSize {
			s.log.Debug(
				"listener",
				"message", "All data read from connection",
				"i", i,
				"service_id", serviceID,
				"connID", conn.id)
		}
	}
}
