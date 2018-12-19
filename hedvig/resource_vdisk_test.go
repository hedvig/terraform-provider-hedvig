package hedvig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccHedvigVdisk(t *testing.T) {
	namevdisk1 := genRandomVdiskName()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHedvigVdiskDestroy(namevdisk1),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccHedvigVdiskConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckHedvigVdiskExists(namevdisk1),
					testAccCheckHedvigVdiskExists("hedvig_vdisk.test-vdisk2"),
					testAccCheckHedvigVdiskSize(namevdisk1),
				),
			},
		},
	})
}

// TODO: Add update vdisk test

var testAccHedvigVdiskConfig = fmt.Sprintf(`
provider "hedvig" {
  node = "%s"
  username = "%s"
  password = "%s"
}

resource "hedvig_vdisk" "test-vdisk1" {
  name = "%s"
  size = 9
  type = "BLOCK"
}

resource "hedvig_vdisk" "test-vdisk2" {
  name = "%s"
  size = 11
  type = "NFS"
}
`, os.Getenv("HV_TESTNODE"), os.Getenv("HV_TESTUSER"), os.Getenv("HV_TESTPASS"),
	genRandomVdiskName(),
	genRandomVdiskName())

func testAccCheckHedvigVdiskExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No vdisk ID is set")
		}

		return nil
	}
}

func testAccCheckHedvigVdiskDestroy(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "hedvig_vdisk" {
				continue
			}
			name := rs.Primary.ID
			if name == n {
				return fmt.Errorf("Found resource: %s", name)
			}
		}
		u := url.URL{}
		u.Host = "tfhashicorp1.external.hedviginc.com"
		u.Path = "/rest/"
		u.Scheme = "http"

		q := url.Values{}
		q.Set("request", fmt.Sprintf("{type:Login,category:UserManagement,params:{userName:'%s',password:'%s',cluster:''}}", os.Getenv("HV_TESTUSER"), os.Getenv("HV_TESTPASS")))
		u.RawQuery = q.Encode()

		resp, err := http.Get(u.String())
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		login := LoginResponse{}
		err = json.Unmarshal(body, &login)

		if err != nil {
			return err
		}

		if login.Status != "ok" {
			return errors.New(login.Message)
		}

		sessionId := login.Result.SessionID

		//return nil

		q = url.Values{}
		q.Set("request", fmt.Sprintf("{type:VirtualDiskDetails,category:VirtualDiskManagement,params{virtualDisk:'vDiskA'},sessionId:'%s'", sessionId))

		u.RawQuery = q.Encode()
		resp, err = http.Get(u.String())
		if err != nil {
			return err
		}

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		readResp := readDiskResponse{}

		err = json.Unmarshal(body, &readResp)

		if err != nil {
			if e, ok := err.(*json.SyntaxError); ok {
				return fmt.Errorf("syntax error at byte offset %d", e.Offset)
			}
			returnable := fmt.Errorf("response: %q", readResp)
			return returnable
		}

		if readResp.Status == "warning" { // && strings.Contains(readResp.Message, "t be found") {
			return nil
		}
		if readResp.Status == "ok" {
			return errors.New("Vdisk still exists")
		}
		return nil //fmt.Errorf("Unknown error: %s", readResp.Status)
	}
}

func testAccCheckHedvigVdiskSize(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.Attributes["size"] == "9" {
			return nil
		}
		if rs.Primary.Attributes["size"] == "" {
			return errors.New("Size expected to not be nil")
		}
		return errors.New("Unknown problem with size of vdisk")
	}
}
