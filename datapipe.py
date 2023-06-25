import sys
import socket
import select
import time

MAXCLIENTS = 20
IDLETIMEOUT = 300

def main():
    # if len(sys.argv) != 5:
    #     print(f"Usage: {sys.argv[0]} localhost localport remotehost remoteport")
    #     return 30

    clients = []
    for _ in range(MAXCLIENTS):
        clients.append({'inuse': False, 'csock': None, 'osock': None, 'activity': None})

    # laddr = (sys.argv[1], int(sys.argv[2]))
    # oaddr = (sys.argv[3], int(sys.argv[4]))
    laddr = ("0.0.0.0", 8080)
    oaddr = ("192.168.0.1", 80)

    lsock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    lsock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    lsock.bind(laddr)
    lsock.listen(5)

    # Create a client socket connected to the outgoing host/port
    osock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    osock.bind(laddr)
    osock.connect(oaddr)

    clients[0]['inuse'] = True
    clients[0]['csock'] = None
    clients[0]['osock'] = osock
    clients[0]['activity'] = time.time()

    while True:
        fdsr, _, _ = select.select([lsock] + [client['csock'] for client in clients if client['inuse']],
                                  [client['osock'] for client in clients if client['inuse']], [], 1.0)

        now = time.time()

        # Check if there are new connections to accept
        if lsock in fdsr:
            csock, _ = lsock.accept()

            for client in clients:
                if not client['inuse']:
                    # Connect the client socket to the outgoing host/port
                    try:
                        osock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
                        osock.bind(laddr)
                        osock.connect(oaddr)
                    except Exception as e:
                        print(f"Failed to connect to the target host: {e}")
                        csock.close()
                        break

                    client['inuse'] = True
                    client['csock'] = csock
                    client['osock'] = osock
                    client['activity'] = now
                    break
            else:
                print("Too many clients")
                csock.close()

        # Service any client connections that have waiting data
        for client in clients:
            if not client['inuse']:
                continue

            closeneeded = False
            if client['csock'] in fdsr:
                try:
                    data = client['csock'].recv(4096)
                    if len(data) <= 0 or client['osock'].send(data) <= 0:
                        closeneeded = True
                    else:
                        client['activity'] = now
                except Exception:
                    closeneeded = True
            elif client['osock'] in fdsr:
                try:
                    data = client['osock'].recv(4096)
                    if len(data) <= 0 or client['csock'].send(data) <= 0:
                        closeneeded = True
                    else:
                        client['activity'] = now
                except Exception:
                    closeneeded = True

            if closeneeded or now - client['activity'] > IDLETIMEOUT:
                client['csock'].close()
                client['osock'].close()
                client['inuse'] = False

if __name__ == "__main__":
    main()
