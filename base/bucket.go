// bucket.go

package base

import (
	"regexp"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/couchbaselabs/walrus"
)

type Bucket walrus.Bucket

// Implementation of walrus.Bucket that talks to a Couchbase server
type couchbaseBucket struct {
	*couchbase.Bucket
}

func (bucket couchbaseBucket) GetName() string {
	return bucket.Name
}

func (bucket couchbaseBucket) Update(k string, exp int, callback walrus.UpdateFunc) error {
	return bucket.Bucket.Update(k, exp, couchbase.UpdateFunc(callback))
}

func (bucket couchbaseBucket) View(ddoc, name string, params map[string]interface{}) (walrus.ViewResult, error) {
	vres := walrus.ViewResult{}
	return vres, bucket.Bucket.ViewCustom(ddoc, name, params, &vres)
}

// Creates a Bucket that talks to a real live Couchbase server.
func GetCouchbaseBucket(couchbaseURL, poolName, bucketName string) (bucket Bucket, err error) {
	cbbucket, err := couchbase.GetBucket(couchbaseURL, poolName, bucketName)
	if err == nil {
		bucket = couchbaseBucket{cbbucket}
	}
	return
}

func GetBucket(url, poolName, bucketName string) (bucket Bucket, err error) {
	if isWalrus, _ := regexp.MatchString(`^(walrus:|file:|/|\.)`, url); isWalrus {
		Log("Opening Walrus database %s on <%s>", bucketName, url)
		walrus.Logging = true
		return walrus.GetBucket(url, poolName, bucketName)
	}
	Log("Opening Couchbase database %s on <%s>", bucketName, url)
	return GetCouchbaseBucket(url, poolName, bucketName)
}
