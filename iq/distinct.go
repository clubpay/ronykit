package iq

// Distinct method returns distinct elements from a collection. The result is an
// unordered collection that contains no duplicate values.
func (q Query[T]) Distinct() Query[T] {
	return Query[T]{
		Iterate: func() Iterator[T] {
			next := q.Iterate()
			set := make(map[T]bool)

			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					if _, has := set[item]; !has {
						set[item] = true
						return
					}
				}

				return
			}
		},
	}
}

//// Distinct method returns distinct elements from a collection. The result is an
//// ordered collection that contains no duplicate values.
////
//// NOTE: Distinct method on OrderedQuery[T] type has better performance than
//// Distinct method on Query[T] type.
//func (oq OrderedQuery[T]) Distinct() OrderedQuery[T] {
//	return OrderedQuery[T]{
//		orders: oq.orders,
//		Query[T]: Query[T]{
//			Iterate: func() Iterator {
//				next := oq.Iterate()
//				var prev interface{}
//
//				return func() (item interface{}, ok bool) {
//					for item, ok = next(); ok; item, ok = next() {
//						if item != prev {
//							prev = item
//							return
//						}
//					}
//
//					return
//				}
//			},
//		},
//	}
//}

// DistinctBy method returns distinct elements from a collection. This method
// executes a selector function for each element to determine a value to compare.
// The result is an unordered collection that contains no duplicate values.
func (q Query[T]) DistinctBy(selector func(T) any) Query[T] {
	return Query[T]{
		Iterate: func() Iterator[T] {
			next := q.Iterate()
			set := make(map[any]bool)

			return func() (item T, ok bool) {
				for item, ok = next(); ok; item, ok = next() {
					s := selector(item)
					if _, has := set[s]; !has {
						set[s] = true
						return
					}
				}

				return
			}
		},
	}
}
