package evaluation

import (
	"context"
	"testing"
)

func TestLabeledRetrievalEvaluation(t *testing.T) {
	r, err := Run(context.Background(), "../../testdata/evaluation")
	if err != nil {
		t.Fatal(err)
	}
	if r.Queries != 60 || r.Corpus != 20 || r.CitationAccuracy != 1 {
		t.Fatal(r)
	}
	t.Logf("Recall@5 %.3f; precision@5 %.3f; cases %d", r.Recall, r.Precision, r.Queries)
}
