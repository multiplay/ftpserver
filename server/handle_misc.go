package server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func (c *clientHandler) handleAUTH() error {
	tlsConfig, err := c.server.driver.GetTLSConfig()
	if err != nil {
		return c.writeMessage(StatusActionNotTaken, fmt.Sprintf("Cannot get a TLS config: %v", err))
	}

	if err := c.writeMessage(StatusAuthAccepted, "AUTH command ok. Expecting TLS Negotiation."); err != nil {
		return err
	}

	c.conn = tls.Server(c.conn, tlsConfig)
	c.reader = bufio.NewReader(c.conn)

	return nil
}

func (c *clientHandler) handlePROT() error {
	// P for Private, C for Clear
	c.transferTLS = c.param == "P"
	return c.writeMessage(StatusOK, "OK")
}

func (c *clientHandler) handlePBSZ() error {
	return c.writeMessage(StatusOK, "Whatever")
}

func (c *clientHandler) handleSYST() error {
	return c.writeMessage(StatusSystemType, "UNIX Type: L8")
}

func (c *clientHandler) handleSTAT() error {
	if c.param == "" {
		// Without a file, it's the server stat
		return c.handleSTATServer()
	}

	// With a file/dir it's the file or the dir's files stat
	return c.handleSTATFile()
}

func (c *clientHandler) handleSITE() error {
	spl := strings.SplitN(c.param, " ", 2)
	if len(spl) > 1 && strings.ToUpper(spl[0]) == "CHMOD" {
		return c.handleCHMOD(spl[1])
	}
	return c.writeMessage(StatusSyntaxErrorParameters, "Not understood SITE subcommand")
}

func (c *clientHandler) handleSTATServer() error {
	if err := c.writeLine(fmt.Sprintf("%v- FTP server status:", StatusSystemStatus)); err != nil {
		return err
	}
	duration := time.Now().UTC().Sub(c.connectedAt)
	duration -= duration % time.Second

	if err := c.writeLine(fmt.Sprintf("Connected to %s from %s for %s", c.server.settings.ListenAddr, c.conn.RemoteAddr(), duration)); err != nil {
		return err
	}
	if c.user != "" {
		if err := c.writeLine(fmt.Sprintf("Logged in as %s", c.user)); err != nil {
			return err
		}
	} else {
		if err := c.writeLine("Not logged in yet"); err != nil {
			return err
		}
	}
	if err := c.writeLine("ftpserver - golang FTP server"); err != nil {
		return err
	}
	return c.writeMessage(StatusSystemStatus, "End")
}

func (c *clientHandler) handleOPTS() error {
	args := strings.SplitN(c.param, " ", 2)
	if strings.ToUpper(args[0]) == "UTF8" {
		return c.writeMessage(StatusOK, "I'm in UTF8 only anyway")
	}
	return c.writeMessage(StatusSyntaxErrorParameters, "Don't know this option")
}

func (c *clientHandler) handleNOOP() error {
	return c.writeMessage(StatusOK, "OK")
}

func (c *clientHandler) handleFEAT() error {
	if err := c.writeLine(fmt.Sprintf("%v- These are my features", StatusSystemStatus)); err != nil {
		return err
	}

	features := []string{
		"UTF8",
		"SIZE",
		"MDTM",
		"REST STREAM",
	}

	if !c.server.settings.DisableMLSD {
		features = append(features, "MLSD")
	}

	if !c.server.settings.DisableMLST {
		features = append(features, "MLST")
	}

	for _, f := range features {
		if err := c.writeLine(" " + f); err != nil {
			return err
		}
	}
	return c.writeMessage(StatusSystemStatus, "End")
}

func (c *clientHandler) handleTYPE() error {
	switch c.param {
	case "I":
		return c.writeMessage(StatusOK, "Type set to binary")
	case "A":
		return c.writeMessage(StatusOK, "WARNING: ASCII isn't correctly supported")
	default:
		return c.writeMessage(StatusSyntaxErrorParameters, "Not understood")
	}
}

func (c *clientHandler) handleQUIT() error {
	if err := c.writeMessage(StatusClosingControlConn, "Goodbye"); err != nil {
		return err
	}
	c.logger.WithFields(logrus.Fields{logKeyAction: "ftp.disconnect", "clean": true}).Debug("Clean disconnect")
	c.Close()

	return nil
}
