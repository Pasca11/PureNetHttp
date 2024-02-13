package storage

import "fmt"

type Item struct {
	Id   int
	Info string
}

type Storage struct {
	Items  map[int]Item
	nextId int
}

func New() *Storage {
	s := &Storage{nextId: 0}
	s.Items = make(map[int]Item)
	return s
}

func (s *Storage) AddItem(info string) int {
	s.nextId++
	s.Items[s.nextId] = Item{Id: s.nextId, Info: info}
	return s.nextId
}

func (s *Storage) GetItem(id int) (Item, error) {
	if i, ok := s.Items[id]; ok {
		return i, nil
	} else {
		return Item{}, fmt.Errorf("failed to get item %d", id)
	}
}

func (s *Storage) DeleteItem(id int) error {
	if _, ok := s.Items[id]; ok {
		delete(s.Items, id)
		return nil
	} else {
		return fmt.Errorf("no such item (id = %d)", id)
	}
}

func (s *Storage) DeleteAll() {
	s.Items = make(map[int]Item)
}

func (s *Storage) GetAll() []Item {
	res := make([]Item, 0, len(s.Items))

	for _, v := range s.Items {
		res = append(res, v)
	}
	return res
}

func (s *Storage) GetItemByInfo(info string) Item {
	for _, v := range s.Items {
		if v.Info == info {
			return v
		}
	}
	return Item{}
}
