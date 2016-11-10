# irc

irc implements the IRC protocol to facilitate communication between clients via a server over TCP. This project contains a server program and a client program which are intended to be used together. Future versions may allow communication with other IRC servers or clients.

# installation

To build this project, make sure you have the latest version of Go installed, then in a console enter:

	go install -v all

# server

To host a server on your localhost, first build the project. Then, navigate to $GOPATH/bin and run server.exe.

# client

To host a client on your localhost, first build the project. Then, navigate to $GOPATH/bin and run client.exe. Make sure a server program is running prior to starting any clients.
