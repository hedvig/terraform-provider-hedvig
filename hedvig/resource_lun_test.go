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

func TestAccHedvigLun(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckHedvigLunDestroy("hedvig_lun.test-lun"),
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccHedvigLunConfig,
				Check:  testAccCheckHedvigLunExists("hedvig_lun.test-lun"),
			},
		},
	})
}

var testAccHedvigLunConfig = fmt.Sprintf(`
provider "hedvig" {
  node = "%s"
  username = "%s"
  password = "%s"
}

resource "hedvig_vdisk" "test-lun-vdisk" {
  name = "%s"
  size = 9
  type = "BLOCK"
}

resource "hedvig_lun" "test-lun" {
  vdisk = "${hedvig_vdisk.test-lun-vdisk.name}"
  controller = "%s"
}
`, os.Getenv("HV_TESTNODE"), os.Getenv("HV_TESTUSER"), os.Getenv("HV_TESTPASS"),
	genRandomVdiskName(),
	os.Getenv("HV_TESTCONT"))

func testAccCheckHedvigLunExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return errors.New("No lun ID is set")
		}

		return nil
	}
}

func testAccCheckHedvigLunDestroy(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "hedvig_lun" {
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

		q = url.Values{}
		q.Set("request", fmt.Sprintf("{type:VirtualDiskDetails,category:VirtualDiskManagement,params{virtualDisk:'%s'},sessionId:'%s'}", n, sessionId))

		u.RawQuery = q.Encode()

		resp, err = http.Get(u.String())
		if err != nil {
			return err
		}

		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		readResp := readLunResponse{}
		err = json.Unmarshal(body, &readResp)
		if err != nil {
			return err
		}

		if readResp.Status == "ok" {
			return errors.New("Error: Lun still exists")
		}
		return nil
	}
}
