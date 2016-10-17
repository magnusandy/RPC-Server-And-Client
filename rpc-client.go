package main

//import "net"
import "fmt"
import "bufio"
import "os"
import "strings"
import "net/rpc"
import "errors"

var stayAlive bool = true;
var myName = "";

//COMMANDS
const COMMAND_PREFIX string = "/";
const HELP_COMMAND string = COMMAND_PREFIX+"help";
const QUIT_COMMAND string = COMMAND_PREFIX+"quit";
const CREATE_ROOM_COMMAND string = COMMAND_PREFIX+"createRoom"; //creates a room with the name of the first argument given
const LIST_ROOMS_COMMAND string = COMMAND_PREFIX+"listRooms"
const JOIN_ROOM_COMMAND string = COMMAND_PREFIX+"join";//   /join roomname will add a user to a rooms list of clients and switch the user to that room
const CURR_ROOM_COMMAND string = COMMAND_PREFIX+"currentRoom";
const CURR_ROOM_USERS_COMMAND string = COMMAND_PREFIX+"currentUsers";
const LEAVE_ROOM_COMMAND string = COMMAND_PREFIX+"leaveRoom";

//Handles the input sent back to the client from the server, simply writes it to the console
//
func getFromServer(conn *rpc.Client){
  for{
    var message string;

    err := conn.Call("Server.MessageClient",&myName, &message)
    if err != nil{
      fmt.Println(err)
      return
    }
    fmt.Print(message)
  }
}
func checkForCommand(message string, clientName string) {
}

type CreateRoomArgs struct{
  ClientName string;
  RoomName string;
}
//Handles user input, reads from stdin and then posts that line to the server
func getfromUser(conn *rpc.Client){
    for{
      reader := bufio.NewReader(os.Stdin)
      message, _ := reader.ReadString('\n')
      var reply string;
      var err error;
      message = strings.TrimSpace(message);//strips the newlines from the string
      isCommand := strings.HasPrefix(message, COMMAND_PREFIX);//checks to see if the line starts with /
      if(isCommand){
        //parse command line, commands should be in the exact form of "/command arg arg arg" where args are not required
        parsedCommand := strings.Split(message, " ")
        if parsedCommand[0] == HELP_COMMAND {
          err = conn.Call("Server.ProcessHelpCommand", &myName, &reply)
        } else if parsedCommand[0] == QUIT_COMMAND {
          stayAlive = false;
        } else if parsedCommand[0] == CREATE_ROOM_COMMAND {
          // not enough arguments to the command
          if len(parsedCommand) < 2{
            err = errors.New("not enough args for create")
          }else{
            args := CreateRoomArgs{myName,parsedCommand[1]}
            err = conn.Call("Server.ServerProcessCreateRoomCommand", &args, &reply)
          }
        } else if parsedCommand[0] == LIST_ROOMS_COMMAND {
          //processListRoomsCommand(client);
        } else if parsedCommand[0] == JOIN_ROOM_COMMAND {
          //not enough given to the command
          if len(parsedCommand) < 2{
            //client.messageClientFromServer(NO_ROOM_NAME_GIVEN_ERR)
          }else{
            //processJoinRoomCommand(client, parsedCommand[1]);
          }
        } else if parsedCommand[0] == CURR_ROOM_COMMAND {
          //processCurrRoomCommand(client);
        }else if parsedCommand[0] == CURR_ROOM_USERS_COMMAND{
        //  processCurrRoomUsersCommand(client);
        }else if parsedCommand[0] == LEAVE_ROOM_COMMAND{
        //  processLeaveRoomCommand(client)
        }

      }else { // message is not a command
      //  sendMessageToCurrentRoom(client, message);
      }

      if err != nil{
        fmt.Println(err)
      }
    }
  }

//starts up the client, starts the recieving thread and the input threads and then loops forever
func main() {

arguments := os.Args[1:];
IP := "localhost";
PORT:= "8080";
if len(arguments) == 0 {
  //no arguments start on localhost 8080
} else if len(arguments) != 2 {
  fmt.Println("I cannot understand your arguments, you must specify no arguments or exactly 2, first the IP and the second as the port")
  return
} else if len(arguments) == 2 {
//correct ammount of args
IP = arguments[0]
PORT = arguments[1]
}
//fmt.Println(arg)
  // connect to this socket
  fmt.Println("Attempting to connect to "+IP+":"+PORT)
  conn, err := rpc.Dial("tcp", IP+":"+PORT)
  if err != nil{
    fmt.Println("Something went wrong with the connection, check that the server exists and that your IP/Port are correct:\nError Message: ")
    fmt.Println(err)
    return
  }

  conn.Call("Server.ConnectMe", "", &myName)
  fmt.Println("Your Username is: "+myName)
  go getfromUser(conn);
  go getFromServer(conn)
  for stayAlive {
    //loops  forever until stayAlive is set to false and then it shuts down
  }
}
