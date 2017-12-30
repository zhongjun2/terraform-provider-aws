package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGameliftVpcPeeringAuthorization() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftVpcPeeringAuthorizationCreate,
		Read:   resourceAwsGameliftVpcPeeringAuthorizationRead,
		Delete: resourceAwsGameliftVpcPeeringAuthorizationDelete,

		Schema: map[string]*schema.Schema{
			"gamelift_aws_account_id": {
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
			"vpc_aws_account_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsGameliftVpcPeeringAuthorizationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	accId := d.Get("gamelift_aws_account_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	input := gamelift.CreateVpcPeeringAuthorizationInput{
		GameLiftAwsAccountId: aws.String(accId),
		PeerVpcId:            aws.String(peerVpcId),
	}
	log.Printf("[INFO] Creating Gamelift VPC Peering Authorization: %s", input)
	_, err := conn.CreateVpcPeeringAuthorization(&input)
	if err != nil {
		return err
	}

	d.SetId(accId + "/" + peerVpcId)

	return resourceAwsGameliftVpcPeeringAuthorizationRead(d, meta)
}

func resourceAwsGameliftVpcPeeringAuthorizationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	accId := d.Get("gamelift_aws_account_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	log.Printf("[INFO] Describing Gamelift VPC Peering Authorization: %s", d.Id())
	out, err := conn.DescribeVpcPeeringAuthorizations(&gamelift.DescribeVpcPeeringAuthorizationsInput{})
	if err != nil {
		return err
	}

	auth := findGameliftVpcPeeringAuth(out.VpcPeeringAuthorizations, accId, peerVpcId)
	if auth != nil {
		log.Printf("[WARN] Gamelift VPC Peering Authorization (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("peer_vpc_aws_account_id", auth.PeerVpcAwsAccountId)

	return nil
}

func findGameliftVpcPeeringAuth(auths []*gamelift.VpcPeeringAuthorization, accId, peerVpcId string) *gamelift.VpcPeeringAuthorization {
	for _, auth := range auths {
		if *auth.GameLiftAwsAccountId == accId && *auth.PeerVpcId == peerVpcId {
			return auth
		}
	}
	return nil
}

func resourceAwsGameliftVpcPeeringAuthorizationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	accId := d.Get("aws_account_id").(string)
	peerVpcId := d.Get("peer_vpc_id").(string)

	log.Printf("[INFO] Deleting Gamelift VPC Peering Authorization: %s", d.Id())
	_, err := conn.DeleteVpcPeeringAuthorization(&gamelift.DeleteVpcPeeringAuthorizationInput{
		GameLiftAwsAccountId: aws.String(accId),
		PeerVpcId:            aws.String(peerVpcId),
	})
	if err != nil {
		return err
	}

	return nil
}
