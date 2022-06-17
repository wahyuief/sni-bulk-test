package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dixonwille/wmenu/v5"
	"github.com/gocolly/colly"
)

type userInput struct {
	option wmenu.Opt
}

func (u *userInput) optFunc(option wmenu.Opt) error {
	u.option = option
	return nil
}

func createMenu(p string, m []string, u *userInput) {
	menu := wmenu.NewMenu(p)
	menu.ChangeReaderWriter(os.Stdin, os.Stdout, os.Stderr)
	for i, m := range m {
		menu.Option(m, i, false, u.optFunc)
		menu.LoopOnInvalid()
		menu.ClearOnMenuRun()
	}

	err := menu.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func ScrapingFromCRTSH(domain string) {
	var hostArr []string
	c := colly.NewCollector(
		colly.AllowedDomains("crt.sh"),
	)

	pageCount := 0
	c.OnRequest(func(r *colly.Request) {
		r.Ctx.Put("url", r.URL.String())
	})

	c.OnHTML("tr td:nth-of-type(5)", func(e *colly.HTMLElement) {
		if !stringInSlice(strings.Trim(strings.ToLower(e.Text), "*."), hostArr) {
			hostArr = append(hostArr, strings.Trim(strings.ToLower(e.Text), "*."))
		}

		f, err := os.Create("data/" + domain + ".txt")
		f, err = os.OpenFile("data/"+domain+".txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		for _, h := range hostArr {
			if _, err := f.WriteString(h + "\n"); err != nil {
				panic(err)
			}
		}
	})

	c.OnResponse(func(r *colly.Response) {
		pageCount++
		log.Println(fmt.Sprintf("[DONE] Domain Scraping From %s\n", r.Request.URL))
	})

	c.Visit("https://crt.sh/?CN=" + domain)
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if a == b {
			return true
		}
	}
	return false
}

func Exists(name string) bool {
	_, err := os.Stat(name)

	if err == nil {
		return true
	}

	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return false
}

func DomainScraping() {
	fmt.Print("Enter domain: ")
	domain := bufio.NewScanner(os.Stdin)
	domain.Scan()

	actFunc := func(opts []wmenu.Opt) error {
		if opts[0].ID == 0 {
			ScrapingFromCRTSH(domain.Text())
		}
		return nil
	}

	if !Exists("data/" + domain.Text() + ".txt") {
		ScrapingFromCRTSH(domain.Text())
	} else {
		confirm := wmenu.NewMenu("The " + domain.Text() + " data is already exist, want to rescraping?")
		confirm.Action(actFunc)
		confirm.IsYesNo(0)
		err := confirm.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func SNIBulkTest() {
	var success string
	fmt.Print("Enter domain: ")
	domain := bufio.NewScanner(os.Stdin)
	domain.Scan()
	file, err := os.Open("data/" + domain.Text() + ".txt")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	host := bufio.NewScanner(file)
	for host.Scan() {
		conf := &tls.Config{
			InsecureSkipVerify: true,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		d := tls.Dialer{
			Config: conf,
		}

		conn, err := d.DialContext(ctx, "tcp", host.Text()+":443")
		cancel()
		if err != nil {
			fmt.Println("Host:", host.Text(), " - ", err)
			continue
		}

		defer conn.Close()

		tlsConn := conn.(*tls.Conn)
		certs := tlsConn.ConnectionState().PeerCertificates
		for _, cert := range certs {
			fmt.Println("Host:", host.Text(), "Issuer:", cert.Issuer)
		}

		success += host.Text() + " - " + fmt.Sprint(certs[0].Issuer) + "\n"
		f, err := os.Create("data/" + domain.Text() + ".success.txt")
		f, err = os.OpenFile("data/"+domain.Text()+".success.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}

		defer f.Close()
		if _, err = f.WriteString(success); err != nil {
			panic(err)
		}
	}

	if err := host.Err(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	prompt := "Choose an option: "
	menuitems := []string{"Domain Scraping", "SNI Bulk Test"}
	u := &userInput{}
	createMenu(prompt, menuitems, u)

	if u.option.ID == 0 {
		DomainScraping()
	}

	if u.option.ID == 1 {
		SNIBulkTest()
	}
}
