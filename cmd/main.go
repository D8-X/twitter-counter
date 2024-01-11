package main

import (
	"fmt"

	"github.com/D8-X/twitter-referral-system/src/svc"
)

func main() {

	fmt.Println("This is social graph service")
	svc.RunTwitterSocialGraphService()
}
