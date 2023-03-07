package simple

import (
	"fmt"
	apiappsv1 "k8s.io/api/apps/v1"
	"strings"
)

const (
	TAG_APPLICATION = "APPLICATION_TAG"
)

func getTagsFromDeployment(dpl *apiappsv1.Deployment)([]string, error){
	if len(dpl.Spec.Template.Spec.Containers) != 1 {
		return nil, fmt.Errorf("found more than one containers in deployment %d", dpl.GetName())
	}

	for _,env := range dpl.Spec.Template.Spec.Containers[0].Env{
		if env.Name == TAG_APPLICATION{
			tags := strings.Split(env.Value, "|")
			return tags, nil
		}
	}
	return nil, nil
}
