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

//continusly asks the server for input by calling the servers message function
func getFromServer(conn *rpc.Client){
  for{
    var message string;//message for return, will be populated by the server
    err := conn.Call("Server.MessageClient",&myName, &message)
    if err != nil{
      fmt.Println("Problem getting messages from the server")
      fmt.Println(err);
      stayAlive = false;
      return
    }
    fmt.Print(message)
  }
}


type DoubleArgs struct{
  Arg1 string;
  Arg2 string;
}

//Handles user input, reads from stdin and then posts that line to the server
func getfromUser(conn *rpc.Client){
    for stayAlive{
      reader := bufio.NewReader(os.Stdin)
      message, _ := reader.ReadString('\n')//read from stdin till the next newline
      var reply string;
      var err error;
      message = strings.TrimSpace(message);//strips the newlines from the input
      isCommand := strings.HasPrefix(message, COMMAND_PREFIX);//checks to see if the line starts with /
      if(isCommand){
        //parse command line, commands should be in the exact form of "/command arg arg arg" where args are not required
        parsedCommand := strings.Split(message, " ")
        if parsedCommand[0] == HELP_COMMAND {
          err = conn.Call("Server.ProcessHelpCommand", &myName, &reply)
        } else if parsedCommand[0] == QUIT_COMMAND {
          err = conn.Call("Server.ProcessQuitCommand", &myName, &reply)
          stayAlive = false;
        } else if parsedCommand[0] == CREATE_ROOM_COMMAND {
          // not enough arguments to the command
          if len(parsedCommand) < 2{
            err = errors.New("not enough args for create room")
          }else{
            args := DoubleArgs{myName,parsedCommand[1]}
            err = conn.Call("Server.ProcessCreateRoomCommand", &args, &reply)
          }
        } else if parsedCommand[0] == LIST_ROOMS_COMMAND {
          err = conn.Call("Server.ProcessListRoomsCommand", &myName, &reply)
        } else if parsedCommand[0] == JOIN_ROOM_COMMAND {
          //not enough given to the command
          if len(parsedCommand) < 2{
            err = errors.New("You must specify a room to join")
          }else{
            args := DoubleArgs{myName,parsedCommand[1]}
            err = conn.Call("Server.ProcessJoinRoomCommand", &args, &reply);
          }
        } else if parsedCommand[0] == CURR_ROOM_COMMAND {
          err = conn.Call("Server.ProcessCurrRoomCommand", &myName, &reply)
        }else if parsedCommand[0] == CURR_ROOM_USERS_COMMAND{
          err = conn.Call("Server.ProcessCurrRoomUsersCommand", &myName, &reply)
        }else if parsedCommand[0] == LEAVE_ROOM_COMMAND{
          err = conn.Call("Server.ProcessLeaveRoomCommand", &myName, &reply)
        }

      }else if stayAlive{ // message is not a command
        args := DoubleArgs{myName,message}
        err = conn.Call("Server.SendMessageToCurrentRoom", &args, &reply);
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
  //call the servers connect function, this will set the client up on the server and return the clients unique Username
  connectErr := conn.Call("Server.Connect", "", &myName)
  if connectErr != nil{
    fmt.Println(connectErr)
    return
  }
  fmt.Println("Your Username is: "+myName)
  go getfromUser(conn);
  go getFromServer(conn)
  for stayAlive {
    //loops  forever until stayAlive is set to false and then it shuts down
  }
}
