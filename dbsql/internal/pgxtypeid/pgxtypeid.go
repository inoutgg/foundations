package pgxtypeid

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"go.jetify.com/typeid/v2"
)

var (
	_ pgtype.TextValuer  = (*TypeID)(nil)
	_ pgtype.TextScanner = (*TypeID)(nil)
)

type TypeID typeid.TypeID

func (id *TypeID) ScanText(v pgtype.Text) error {
	if !v.Valid {
		return fmt.Errorf("pgxtypeid: cannot scan NULL into *typeid.TypeID")
	}

	val, err := typeid.Parse(v.String)
	if err != nil {
		return fmt.Errorf("pgxtypeid: cannot parse string: %w", err)
	}

	*id = TypeID(val)
	return nil
}

func (id TypeID) TextValue() (pgtype.Text, error) {
	return pgtype.Text{
		String: typeid.TypeID(id).String(),
		Valid:  true,
	}, nil
}

func TryWrapTypeIDEncodePlan(
	value any,
) (plan pgtype.WrappedEncodePlanNextSetter, nextValue any, ok bool) {
	switch value := value.(type) {
	case typeid.TypeID:
		return &wrapTypeIDEncodePlan{}, TypeID(value), true
	}

	return nil, nil, false
}

type wrapTypeIDEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapTypeIDEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapTypeIDEncodePlan) Encode(value any, buf []byte) (newBuf []byte, err error) {
	return plan.next.Encode(TypeID(value.(typeid.TypeID)), buf)
}

func TryWrapTypeIDScanPlan(
	target any,
) (plan pgtype.WrappedScanPlanNextSetter, nextDst any, ok bool) {
	switch target := target.(type) {
	case *typeid.TypeID:
		return &wrapTypeIDScanPlan{}, (*TypeID)(target), true
	}

	return nil, nil, false
}

type wrapTypeIDScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapTypeIDScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapTypeIDScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*TypeID)(dst.(*typeid.TypeID)))
}

func Register(tm *pgtype.Map) {
	tm.TryWrapEncodePlanFuncs = append(
		[]pgtype.TryWrapEncodePlanFunc{TryWrapTypeIDEncodePlan},
		tm.TryWrapEncodePlanFuncs...)
	tm.TryWrapScanPlanFuncs = append(
		[]pgtype.TryWrapScanPlanFunc{TryWrapTypeIDScanPlan},
		tm.TryWrapScanPlanFuncs...)
}
