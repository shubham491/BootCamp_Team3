package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/elgs/gojq"
	"github.com/varungupte/BootCamp_Team3/pkg/errorutil"
	"github.com/varungupte/BootCamp_Team3/pkg/restaurants"
	"github.com/varungupte/BootCamp_Team3/pkg/services/orders/orderspb"
	"github.com/varungupte/BootCamp_Team3/pkg/users"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
)

type Order struct {
	Id int
	Quantity int
	Amount float64
	DishName string
	User users.User
	Restau restaurants.Restaurant
	DeliveryTime string
}

var gJsonData string

func convertToJSON(orders []Order)  {
	jsonData, err := json.Marshal(orders)
	errorutil.CheckError(err, "")

	jsonFile, err := os.Create("./orders.json")
	errorutil.CheckError(err, "")

	defer jsonFile.Close()

	jsonFile.Write(jsonData)
	jsonFile.Close()

	gJsonData = string(jsonData)
}

func GenerateOrdersJSON(filename string) {
	usrs := users.GetUsers("User.csv")
	rests := restaurants.GetRestaurants("Restaurant.csv")

	orderFile, err := os.Open(filename)
	errorutil.CheckError(err, "")

	defer orderFile.Close()

	reader := csv.NewReader(orderFile)
	reader.FieldsPerRecord = -1

	orderData, err := reader.ReadAll()
	errorutil.CheckError(err, "")

	var ord Order
	var orders []Order

	for _, each := range orderData {
		ord.Id,_ = strconv.Atoi(each[0])
		ord.Amount,_ = strconv.ParseFloat((each[1]),32)
		ord.Quantity,_ = strconv.Atoi(each[2])
		ord.DishName= each[3]
		var userid int
		userid,_= strconv.Atoi(each[4])
		ord.User = usrs[userid-1]
		var resid int
		resid,_ = strconv.Atoi(each[5])
		ord.Restau = rests[resid-1]
		ord.DeliveryTime = each[6]
		orders= append(orders, ord)
	}
	convertToJSON(orders)
}

type orders_server struct {

}

func main()  {
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Sorry failed to load server %v: ", err)
	}

	s := grpc.NewServer()

	orderspb.RegisterOrdersServiceServer(s, &orders_server{})
	GenerateOrdersJSON("Orders.csv")

	fmt.Println("Orders Server starting...")
	if s.Serve(lis); err != nil {
		log.Fatalf("failed to Serve %v", err)
	}
}

func (*orders_server) GetOrdersCount(ctx context.Context, req *orderspb.OrdersCountRequest) (*orderspb.OrdersCountResponse, error)  {
	var orders []Order
	err := json.Unmarshal([]byte(gJsonData), &orders)
	if err != nil {
		return nil, err
	}
	res := &orderspb.OrdersCountResponse{
		Count: strconv.Itoa(len(orders)),
	}
	return res, nil
}

func (*orders_server) GetPopularDish(ctx context.Context,req *orderspb.PopularDishRequest) (*orderspb.PopularDishResponse, error) {
	//Using gojq library https://github.com/elgs/gojq#gojq
	parser, _ := gojq.NewStringQuery(gJsonData)
	cityName := req.CityName
	//Popular Dish Areawise (In a particular User City, which is the dish maximum ordered)
	var m = make(map[string]int)
	for i := 0; i < 1000; i++ {
		var f string
		f = "[" + strconv.Itoa(i) + "].User.City"
		q, _ := parser.Query(f)
		if q == cityName {
			var d string
			d = "[" + strconv.Itoa(i) + "].DishName"
			dishName, _ := parser.Query(d)
			m[dishName.(string)] = m[dishName.(string)] + 1
		}
	}

	// Iterating map
	var name string
	maxres := -1
	for i, p := range m {
		if p > maxres {
			name = i
			maxres = p
		}
	}
	res := &orderspb.PopularDishResponse{
		DishName: name,
	}
	return res, nil
}

func (*orders_server) GetOrderDetail (ctx context.Context, req *orderspb.OrderDetailRequest) (*orderspb.OrderDetailResponse, error) {
	orderNumber:= req.OrderNumber

	parser, _ := gojq.NewStringQuery(gJsonData)
	ord,_ := strconv.Atoi(orderNumber)
	ord = ord-1
	quer := "["+strconv.Itoa(ord)+"]"
	orderDetail, _ := parser.Query(quer)

	result := fmt.Sprint(orderDetail)
	res := &orderspb.OrderDetailResponse{
		OrderDetail: result,
	}
	return res, nil
}

func parseJsonFile(jsonFilePath string) ([]Order, error){
	orderJsonFile, err := os.Open(jsonFilePath)
	var orderList []Order

	if err != nil {
		return orderList, err
	}
	defer orderJsonFile.Close()

	byteValue, _ := ioutil.ReadAll(orderJsonFile)
	json.Unmarshal(byteValue, &orderList)

	return orderList, nil
}

func writeJsonFile(jsonFilePath string, ordersList []Order) error{
	jsonData, err := json.Marshal(ordersList)
	if err!= nil {
		return err
	}

	err = ioutil.WriteFile(jsonFilePath, jsonData, 0644)
	if err!= nil {
		return err
	}

	return nil
}

//This function updates the dish of order given order_id it opens file and write to it after update
func (*orders_server) UpdateDish (ctx context.Context, req *orderspb.UpdateDishRequest) (*orderspb.UpdateDishResponse, error) {
	fmt.Println("Update dish called")

	order_id := int(req.GetOrderId())
	updated_dish := req.GetUpdatedDish()
	res := &orderspb.UpdateDishResponse{
		Status: "SUCCESS: Order updated",
	}

	jsonFilePath := "./orders.json"
	orderList, err := parseJsonFile(jsonFilePath)
	if err != nil {
		res = &orderspb.UpdateDishResponse{
			Status: "Failed to write in file",
		}

		return res, err
	}

	for i, order := range orderList {
		if order.Id == order_id {
			log.Println(orderList[i].DishName)
			orderList[i].DishName = updated_dish
			err = writeJsonFile(jsonFilePath, orderList)
			if err!=nil {
				res = &orderspb.UpdateDishResponse{
					Status: "Failed to write in file",
				}
			}

			return res, err
		}
	}

	res = &orderspb.UpdateDishResponse{
		Status: "FAILURE: No order found with this orderId",
	}
	return res, nil
}