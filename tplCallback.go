package florid

import (
	"net/http"
	"fmt"
	
)

func tplCallback(res http.ResponseWriter, req *http.Request) (int) {
	
	fmt.Println("this is tpl");
	return SUCCESS
}