package collections

type OrderedMap struct {
	m    map[string]interface{}
	keys []string
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		m:    make(map[string]interface{}),
		keys: []string{},
	}
}

func (t *OrderedMap) Get(key string) interface{} {
	return t.m[key]
}

func (t *OrderedMap) Put(key string, value interface{}) {
	if _, existed := t.m[key]; !existed {
		t.keys = append(t.keys, key)
	}
	t.m[key] = value
}

func (t *OrderedMap) Keys() []string {
	return t.keys
}
