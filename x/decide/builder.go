package decide

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"github.com/clubpay/ronykit/x/rkit"
)

// Describer can be implemented by T (or *T) so a struct defines its own decision schema
// (Option 2) without passing a builder callback to NewSchemaFor / MustSchemaFor.
type Describer[T any] interface {
	Describe(b *Builder[T])
}

// CriteriaProvider can be implemented by a custom choice field type E (Option 3)
// to supply option labels and their rubric descriptions automatically.
// Any custom type implementing `Criteria() map[E]string` or `Criteria() map[string]string`
// or `Options() []ChoiceOption` or `Options() []E` is also discovered automatically via reflection.
type CriteriaProvider[E comparable] interface {
	Criteria() map[E]string
}

// LevelsProvider can be implemented by a custom score field type (Option 3)
// to supply ordered rubric level descriptions automatically.
type LevelsProvider interface {
	Levels() []string
}

// MinLevelProvider can be optionally implemented alongside LevelsProvider
// to set the minimum integer value corresponding to level index 0 (default 0).
type MinLevelProvider interface {
	MinLevel() int
}

type fieldEntryKey struct {
	offset uintptr
	typ    reflect.Type
}

type structFieldMeta struct {
	indexPath []int
	namePath  string
	snakeID   string
	offset    uintptr
	typ       reflect.Type
}

type fieldPlan struct {
	meta          structFieldMeta
	question      Question
	threshold     float64
	minLevel      int
	discreteScore bool
	rawProb       bool
	minConfidence *float64
	defaultVal    any
	err           error
}

// BuildFunc configures the decision schema on Builder[T] using field pointers from proto.
type BuildFunc[T any] func(b *Builder[T], proto *T)

// Builder records question definitions bound to struct fields of T.
type Builder[T any] struct {
	protoAddr uintptr
	byKey     map[fieldEntryKey]structFieldMeta
	byOffset  map[uintptr][]structFieldMeta
	plans     []*fieldPlan
	errs      []error
}

func newBuilder[T any](proto *T) (*Builder[T], error) {
	rt := reflect.TypeFor[T]()
	if rt.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: schema target type %v must be a struct", ErrInvalidSchema, rt)
	}

	b := &Builder[T]{
		protoAddr: uintptr(unsafe.Pointer(proto)),
		byKey:     make(map[fieldEntryKey]structFieldMeta),
		byOffset:  make(map[uintptr][]structFieldMeta),
	}

	b.indexStruct(rt, nil, "", 0)

	return b, nil
}

func (b *Builder[T]) indexStruct(
	rt reflect.Type,
	indexPrefix []int,
	namePrefix string,
	baseOffset uintptr,
) {
	for i := range rt.NumField() {
		sf := rt.Field(i)
		if !sf.IsExported() {
			continue
		}

		idxPath := make([]int, len(indexPrefix)+1)
		copy(idxPath, indexPrefix)
		idxPath[len(indexPrefix)] = i

		fieldOffset := baseOffset + sf.Offset

		fieldName := sf.Name
		if namePrefix != "" && !sf.Anonymous {
			fieldName = namePrefix + "." + sf.Name
		} else if namePrefix != "" && sf.Anonymous {
			fieldName = namePrefix
		}

		snakeParts := strings.Split(fieldName, ".")
		for idx, part := range snakeParts {
			snakeParts[idx] = rkit.ToSnake(part)
		}

		snakeID := strings.Join(snakeParts, "_")

		meta := structFieldMeta{
			indexPath: idxPath,
			namePath:  fieldName,
			snakeID:   snakeID,
			offset:    fieldOffset,
			typ:       sf.Type,
		}

		b.byKey[fieldEntryKey{offset: fieldOffset, typ: sf.Type}] = meta
		b.byOffset[fieldOffset] = append(b.byOffset[fieldOffset], meta)

		if sf.Type.Kind() == reflect.Struct {
			b.indexStruct(sf.Type, idxPath, fieldName, fieldOffset)
		}
	}
}

func (b *Builder[T]) resolveField(fieldPtr any) (structFieldMeta, error) {
	if fieldPtr == nil {
		return structFieldMeta{}, fmt.Errorf("%w: field pointer must not be nil", ErrInvalidSchema)
	}

	rv := reflect.ValueOf(fieldPtr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return structFieldMeta{}, fmt.Errorf(
			"%w: expected non-nil pointer to struct field, got %T",
			ErrInvalidSchema, fieldPtr,
		)
	}

	ptrAddr := rv.Pointer()
	if ptrAddr < b.protoAddr {
		return structFieldMeta{}, fmt.Errorf(
			"%w: field pointer does not belong to schema prototype struct",
			ErrInvalidSchema,
		)
	}

	offset := ptrAddr - b.protoAddr
	elemType := rv.Type().Elem()

	if meta, ok := b.byKey[fieldEntryKey{offset: offset, typ: elemType}]; ok {
		return meta, nil
	}

	return structFieldMeta{}, fmt.Errorf(
		"%w: pointer of type %v at offset %d does not match any exported field in %v",
		ErrInvalidSchema, rv.Type(), offset, reflect.TypeFor[T](),
	)
}

func (b *Builder[T]) addPlan(meta structFieldMeta, kind QuestionKind, instructions any) *fieldPlan {
	for _, existing := range b.plans {
		if slices.Equal(existing.meta.indexPath, meta.indexPath) {
			existing.question.Kind = kind
			existing.question.Instructions = instructions

			return existing
		}
	}

	p := &fieldPlan{
		meta: meta,
		question: Question{
			ID:           meta.snakeID,
			Kind:         kind,
			Instructions: instructions,
		},
		threshold: DefaultNoulThreshold,
	}
	b.plans = append(b.plans, p)

	return p
}

// Choice binds a string, integer, or custom enum field (or pointer to one) to a KindChoice question.
// If the field's type implements Criteria() or Options(), those options are loaded automatically
// and can be extended or overridden via the returned ChoiceFieldBuilder.
func (b *Builder[T]) Choice(fieldPtr any, instructions any) *ChoiceFieldBuilder {
	meta, err := b.resolveField(fieldPtr)
	if err != nil {
		b.errs = append(b.errs, err)

		return &ChoiceFieldBuilder{plan: &fieldPlan{err: err}}
	}

	targetType := derefType(meta.typ)
	if !isChoiceTargetKind(targetType.Kind()) {
		err := fmt.Errorf(
			"%w: field %s (%v) cannot be bound to Choice; expected string or integer type",
			ErrInvalidSchema, meta.namePath, meta.typ,
		)
		b.errs = append(b.errs, err)

		return &ChoiceFieldBuilder{plan: &fieldPlan{err: err}}
	}

	plan := b.addPlan(meta, KindChoice, instructions)
	if opts := extractTypeOptions(targetType); len(opts) > 0 {
		plan.question.Options = opts
	}

	return &ChoiceFieldBuilder{plan: plan}
}

// Score binds an integer, float, or custom numeric field (or pointer to one) to a KindScore question.
// If the field's type implements LevelsProvider (`Levels() []string`), those levels are loaded
// automatically and can be overridden via the returned ScoreFieldBuilder.
func (b *Builder[T]) Score(fieldPtr any, instructions any) *ScoreFieldBuilder {
	meta, err := b.resolveField(fieldPtr)
	if err != nil {
		b.errs = append(b.errs, err)

		return &ScoreFieldBuilder{plan: &fieldPlan{err: err}}
	}

	targetType := derefType(meta.typ)
	if !isScoreTargetKind(targetType.Kind()) {
		err := fmt.Errorf(
			"%w: field %s (%v) cannot be bound to Score; expected integer or float type",
			ErrInvalidSchema, meta.namePath, meta.typ,
		)
		b.errs = append(b.errs, err)

		return &ScoreFieldBuilder{plan: &fieldPlan{err: err}}
	}

	plan := b.addPlan(meta, KindScore, instructions)
	plan.discreteScore = isIntKind(targetType.Kind())

	if lvls, ok := extractTypeLevels(targetType); ok {
		plan.question.Levels = lvls
	}

	if minLvl, ok := extractTypeMinLevel(targetType); ok {
		plan.minLevel = minLvl
	}

	return &ScoreFieldBuilder{plan: plan}
}

// Bool binds a bool (or *bool / ~bool) field to a KindNoul question thresholded at 0.5 (by default).
func (b *Builder[T]) Bool(fieldPtr any, instructions any) *BoolFieldBuilder {
	meta, err := b.resolveField(fieldPtr)
	if err != nil {
		b.errs = append(b.errs, err)

		return &BoolFieldBuilder{plan: &fieldPlan{err: err}}
	}

	targetType := derefType(meta.typ)
	if targetType.Kind() != reflect.Bool {
		err := fmt.Errorf(
			"%w: field %s (%v) cannot be bound to Bool; expected bool type",
			ErrInvalidSchema, meta.namePath, meta.typ,
		)
		b.errs = append(b.errs, err)

		return &BoolFieldBuilder{plan: &fieldPlan{err: err}}
	}

	plan := b.addPlan(meta, KindNoul, instructions)
	plan.rawProb = false

	return &BoolFieldBuilder{plan: plan}
}

// Prob binds a float64 or float32 (or pointer) field to a KindNoul question, hydrating
// the raw probability in [0, 1] directly into the field.
func (b *Builder[T]) Prob(fieldPtr any, instructions any) *ProbFieldBuilder {
	meta, err := b.resolveField(fieldPtr)
	if err != nil {
		b.errs = append(b.errs, err)

		return &ProbFieldBuilder{plan: &fieldPlan{err: err}}
	}

	targetType := derefType(meta.typ)
	if !isFloatKind(targetType.Kind()) {
		err := fmt.Errorf(
			"%w: field %s (%v) cannot be bound to Prob; expected float32 or float64 type",
			ErrInvalidSchema, meta.namePath, meta.typ,
		)
		b.errs = append(b.errs, err)

		return &ProbFieldBuilder{plan: &fieldPlan{err: err}}
	}

	plan := b.addPlan(meta, KindNoul, instructions)
	plan.rawProb = true

	return &ProbFieldBuilder{plan: plan}
}

// ChoiceFieldBuilder configures a KindChoice field binding.
type ChoiceFieldBuilder struct {
	plan        *fieldPlan
	clearedAuto bool
}

// ID overrides the default snake_case question ID for this field.
func (c *ChoiceFieldBuilder) ID(id string) *ChoiceFieldBuilder {
	if c.plan.err == nil {
		c.plan.question.ID = strings.TrimSpace(id)
	}

	return c
}

// Option appends or updates a single choice option label and optional rubric description.
// Calling Option on a field that loaded automatic type options replaces the automatic list
// with the explicitly chained options.
func (c *ChoiceFieldBuilder) Option(value any, description ...string) *ChoiceFieldBuilder {
	if c.plan.err != nil {
		return c
	}

	if !c.clearedAuto {
		c.plan.question.Options = nil
		c.clearedAuto = true
	}

	label := formatOptionLabel(value)

	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}

	for i := range c.plan.question.Options {
		if c.plan.question.Options[i].Label == label {
			c.plan.question.Options[i].Description = desc

			return c
		}
	}

	c.plan.question.Options = append(c.plan.question.Options, ChoiceOption{
		Label:       label,
		Description: desc,
	})

	return c
}

// Options sets multiple option labels without descriptions.
func (c *ChoiceFieldBuilder) Options(values ...any) *ChoiceFieldBuilder {
	for _, v := range values {
		c.Option(v)
	}

	return c
}

// Criteria sets options from a map[K]string (such as map[string]string or map[MyEnum]string)
// sorted deterministically by label.
func (c *ChoiceFieldBuilder) Criteria(criteriaMap any) *ChoiceFieldBuilder {
	if c.plan.err != nil || criteriaMap == nil {
		return c
	}

	opts, ok := extractMapOptions(reflect.ValueOf(criteriaMap))
	if !ok {
		c.plan.err = fmt.Errorf("%w: Criteria expects map[K]string, got %T", ErrInvalidSchema, criteriaMap)

		return c
	}

	c.clearedAuto = true
	c.plan.question.Options = opts

	return c
}

// MinConfidence sets the minimum AnswerConfidence required to populate this field.
// If the answer's confidence is below minConf, pointer fields remain nil and value fields
// receive the Default value (or zero value).
func (c *ChoiceFieldBuilder) MinConfidence(minConf float64) *ChoiceFieldBuilder {
	if c.plan.err == nil {
		c.plan.minConfidence = &minConf
	}

	return c
}

// Default sets the fallback value used when MinConfidence is not met.
func (c *ChoiceFieldBuilder) Default(val any) *ChoiceFieldBuilder {
	if c.plan.err == nil {
		c.plan.defaultVal = val
	}

	return c
}

// ScoreFieldBuilder configures a KindScore field binding.
type ScoreFieldBuilder struct {
	plan *fieldPlan
}

// ID overrides the default snake_case question ID for this field.
func (s *ScoreFieldBuilder) ID(id string) *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.question.ID = strings.TrimSpace(id)
	}

	return s
}

// Levels sets the ordered rubric level descriptions from lowest to highest (2 to 10 levels).
func (s *ScoreFieldBuilder) Levels(levels ...string) *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.question.Levels = slices.Clone(levels)
	}

	return s
}

// Min sets the integer value corresponding to level index 0 (defaults to 0).
// For example, Min(1) with 5 levels maps level indices 0..4 to values 1..5.
func (s *ScoreFieldBuilder) Min(minVal int) *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.minLevel = minVal
	}

	return s
}

// Discrete forces the field to receive the discrete winning level (Min + argmax)
// even when the target field is a float.
func (s *ScoreFieldBuilder) Discrete() *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.discreteScore = true
	}

	return s
}

// Expected configures a float field to receive the continuous probability-weighted score
// (Min + score) rather than a discrete level index.
func (s *ScoreFieldBuilder) Expected() *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.discreteScore = false
	}

	return s
}

// MinConfidence sets the minimum AnswerConfidence required to populate this field.
func (s *ScoreFieldBuilder) MinConfidence(minConf float64) *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.minConfidence = &minConf
	}

	return s
}

// Default sets the fallback value used when MinConfidence is not met.
func (s *ScoreFieldBuilder) Default(val any) *ScoreFieldBuilder {
	if s.plan.err == nil {
		s.plan.defaultVal = val
	}

	return s
}

// BoolFieldBuilder configures a KindNoul boolean field binding.
type BoolFieldBuilder struct {
	plan *fieldPlan
}

// ID overrides the default snake_case question ID for this field.
func (bf *BoolFieldBuilder) ID(id string) *BoolFieldBuilder {
	if bf.plan.err == nil {
		bf.plan.question.ID = strings.TrimSpace(id)
	}

	return bf
}

// Threshold sets the noul probability threshold in (0, 1) at or above which the bool field
// is hydrated as true (defaults to 0.5).
func (bf *BoolFieldBuilder) Threshold(threshold float64) *BoolFieldBuilder {
	if bf.plan.err == nil {
		bf.plan.threshold = threshold
	}

	return bf
}

// Criteria sets optional descriptions for what true and false mean.
func (bf *BoolFieldBuilder) Criteria(trueDesc, falseDesc string) *BoolFieldBuilder {
	if bf.plan.err == nil {
		bf.plan.question.NoulCriteria = &NoulCriteria{
			True:  trueDesc,
			False: falseDesc,
		}
	}

	return bf
}

// MinConfidence sets the minimum AnswerConfidence required to populate this field.
func (bf *BoolFieldBuilder) MinConfidence(minConf float64) *BoolFieldBuilder {
	if bf.plan.err == nil {
		bf.plan.minConfidence = &minConf
	}

	return bf
}

// Default sets the fallback boolean value used when MinConfidence is not met.
func (bf *BoolFieldBuilder) Default(val bool) *BoolFieldBuilder {
	if bf.plan.err == nil {
		bf.plan.defaultVal = val
	}

	return bf
}

// ProbFieldBuilder configures a KindNoul raw probability field binding.
type ProbFieldBuilder struct {
	plan *fieldPlan
}

// ID overrides the default snake_case question ID for this field.
func (pf *ProbFieldBuilder) ID(id string) *ProbFieldBuilder {
	if pf.plan.err == nil {
		pf.plan.question.ID = strings.TrimSpace(id)
	}

	return pf
}

// Criteria sets optional descriptions for what true and false mean.
func (pf *ProbFieldBuilder) Criteria(trueDesc, falseDesc string) *ProbFieldBuilder {
	if pf.plan.err == nil {
		pf.plan.question.NoulCriteria = &NoulCriteria{
			True:  trueDesc,
			False: falseDesc,
		}
	}

	return pf
}

// MinConfidence sets the minimum AnswerConfidence required to populate this field.
func (pf *ProbFieldBuilder) MinConfidence(minConf float64) *ProbFieldBuilder {
	if pf.plan.err == nil {
		pf.plan.minConfidence = &minConf
	}

	return pf
}

// Default sets the fallback float value used when MinConfidence is not met.
func (pf *ProbFieldBuilder) Default(val float64) *ProbFieldBuilder {
	if pf.plan.err == nil {
		pf.plan.defaultVal = val
	}

	return pf
}

func derefType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t
}

func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloatKind(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

func isChoiceTargetKind(k reflect.Kind) bool {
	return k == reflect.String || isIntKind(k)
}

func isScoreTargetKind(k reflect.Kind) bool {
	return isIntKind(k) || isFloatKind(k)
}

func formatOptionLabel(v any) string {
	if v == nil {
		return ""
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.String:
		return rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10) //rkitcheck:ignore reflect int64 formatting
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10) //rkitcheck:ignore reflect uint64 formatting
	default:
		return fmt.Sprint(v)
	}
}

func extractTypeOptions(t reflect.Type) []ChoiceOption {
	zeroVal := reflect.Zero(t)

	if opts, ok := callOptionMethod(zeroVal); ok {
		return opts
	}

	ptrVal := reflect.New(t)
	if opts, ok := callOptionMethod(ptrVal); ok {
		return opts
	}

	return nil
}

func callOptionMethod(rv reflect.Value) ([]ChoiceOption, bool) {
	if m := rv.MethodByName("Criteria"); m.IsValid() {
		mt := m.Type()
		if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Map {
			out := m.Call(nil)[0]
			if opts, ok := extractMapOptions(out); ok && len(opts) > 0 {
				return opts, true
			}
		}
	}

	if m := rv.MethodByName("Options"); m.IsValid() {
		mt := m.Type()
		if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Slice {
			out := m.Call(nil)[0]
			if opts, ok := extractSliceOptions(out); ok && len(opts) > 0 {
				return opts, true
			}
		}
	}

	return nil, false
}

func extractMapOptions(rv reflect.Value) ([]ChoiceOption, bool) {
	if !rv.IsValid() || rv.Kind() != reflect.Map {
		return nil, false
	}

	if rv.Type().Elem().Kind() != reflect.String {
		return nil, false
	}

	keyKind := rv.Type().Key().Kind()
	if !isChoiceTargetKind(keyKind) {
		return nil, false
	}

	opts := make([]ChoiceOption, 0, rv.Len())

	iter := rv.MapRange()
	for iter.Next() {
		opts = append(opts, ChoiceOption{
			Label:       formatOptionLabel(iter.Key().Interface()),
			Description: iter.Value().String(),
		})
	}

	slices.SortFunc(opts, func(a, b ChoiceOption) int {
		return strings.Compare(a.Label, b.Label)
	})

	return opts, true
}

func extractSliceOptions(rv reflect.Value) ([]ChoiceOption, bool) {
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil, false
	}

	choiceOptType := reflect.TypeFor[ChoiceOption]()
	if rv.Type().Elem() == choiceOptType {
		opts := make([]ChoiceOption, rv.Len())
		for i := range rv.Len() {
			opts[i], _ = reflect.TypeAssert[ChoiceOption](
				rv.Index(i),
			) //rkitcheck:ignore type already verified
		}

		return opts, true
	}

	if !isChoiceTargetKind(rv.Type().Elem().Kind()) {
		return nil, false
	}

	opts := make([]ChoiceOption, rv.Len())
	for i := range rv.Len() {
		opts[i] = ChoiceOption{
			Label: formatOptionLabel(rv.Index(i).Interface()),
		}
	}

	return opts, true
}

func extractTypeLevels(t reflect.Type) ([]string, bool) {
	if v, ok := reflect.TypeAssert[LevelsProvider](reflect.Zero(t)); ok {
		return slices.Clone(v.Levels()), true
	}

	if v, ok := reflect.TypeAssert[LevelsProvider](reflect.New(t)); ok {
		return slices.Clone(v.Levels()), true
	}

	return nil, false
}

func extractTypeMinLevel(t reflect.Type) (int, bool) {
	if v, ok := reflect.TypeAssert[MinLevelProvider](reflect.Zero(t)); ok {
		return v.MinLevel(), true
	}

	if v, ok := reflect.TypeAssert[MinLevelProvider](reflect.New(t)); ok {
		return v.MinLevel(), true
	}

	return 0, false
}
