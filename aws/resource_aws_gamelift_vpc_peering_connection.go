package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGameliftVpcPeeringConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftVpcPeeringConnectionCreate,
		Read:   resourceAwsGameliftVpcPeeringConnectionRead,
		Delete: resourceAwsGameliftVpcPeeringConnectionDelete,

		Schema: map[string]*schema.Schema{
			"fleet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"peer_vpc_aws_account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"peer_vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"gamelift_vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ipv4_cidr_block": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"vpc_peering_connection_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsGameliftVpcPeeringConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	fleetId := d.Get("fleet_id").(string)
	peerVpcAccId := d.Get("peer_vpc_aws_account_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	input := gamelift.CreateVpcPeeringConnectionInput{
		FleetId:             aws.String(fleetId),
		PeerVpcAwsAccountId: aws.String(peerVpcAccId),
		PeerVpcId:           aws.String(peerVpcId),
	}
	log.Printf("[INFO] Creating Gamelift VPC Peering Connection: %s", input)
	_, err := conn.CreateVpcPeeringConnection(&input)
	if err != nil {
		return err
	}

	d.SetId(fleetId + "/" + peerVpcAccId + "/" + peerVpcId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{"deleted", "pending-acceptance", "provisioning"},
		Target:  []string{"active"},
		Timeout: 45 * time.Second,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeVpcPeeringConnections(&gamelift.DescribeVpcPeeringConnectionsInput{
				FleetId: aws.String(fleetId),
			})
			if err != nil {
				return out, "", err
			}

			pConn := findGameliftVpcPeeringConnection(out.VpcPeeringConnections, peerVpcId)
			if pConn == nil {
				return nil, "", nil
			}

			status := *pConn.Status.Code
			var errReason error
			if status == "failed" {
				errReason = fmt.Errorf("Message: %s", *pConn.Status.Message)
			}

			return pConn, status, errReason
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsGameliftVpcPeeringConnectionRead(d, meta)
}

func resourceAwsGameliftVpcPeeringConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	fleetId := d.Get("fleet_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	log.Printf("[INFO] Describing Gamelift VPC Peering Connection: %s", d.Id())
	out, err := conn.DescribeVpcPeeringConnections(&gamelift.DescribeVpcPeeringConnectionsInput{
		FleetId: aws.String(fleetId),
	})
	if err != nil {
		return err
	}

	pConn := findGameliftVpcPeeringConnection(out.VpcPeeringConnections, peerVpcId)
	if pConn == nil {
		log.Printf("[WARN] Gamelift VPC Peering Connection (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("gamelift_vpc_id", pConn.GameLiftVpcId)
	d.Set("ipv4_cidr_block", pConn.IpV4CidrBlock)
	d.Set("vpc_peering_connection_id", pConn.VpcPeeringConnectionId)

	return nil
}

func findGameliftVpcPeeringConnection(conns []*gamelift.VpcPeeringConnection, peerVpcId string) *gamelift.VpcPeeringConnection {
	log.Printf("[DEBUG] Looking for Peering Connection w/ Peer VPC ID %q", peerVpcId)
	for _, conn := range conns {
		log.Printf("[DEBUG] Comparing VPC ID %q to %q", *conn.PeerVpcId, peerVpcId)
		if *conn.PeerVpcId == peerVpcId {
			return conn
		}
	}
	return nil
}

func resourceAwsGameliftVpcPeeringConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	fleetId := d.Get("fleet_id").(string)
	connId := d.Get("vpc_peering_connection_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	log.Printf("[INFO] Deleting Gamelift VPC Peering Connection: %s", d.Id())
	_, err := conn.DeleteVpcPeeringConnection(&gamelift.DeleteVpcPeeringConnectionInput{
		FleetId:                aws.String(fleetId),
		VpcPeeringConnectionId: aws.String(connId),
	})
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Timeout: 15 * time.Second,
		Pending: []string{"active", "pending-acceptance", "provisioning"},
		Target:  []string{"deleted"},
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeVpcPeeringConnections(&gamelift.DescribeVpcPeeringConnectionsInput{
				FleetId: aws.String(fleetId),
			})
			if err != nil {
				return out, "", err
			}

			pConn := findGameliftVpcPeeringConnection(out.VpcPeeringConnections, peerVpcId)
			if pConn == nil {
				return nil, "deleted", nil
			}

			status := *pConn.Status.Code
			var errReason error
			if status == "failed" {
				errReason = fmt.Errorf("Message: %s", *pConn.Status.Message)
			}

			return pConn, status, errReason
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}
