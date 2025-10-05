package payer

type Account struct {
	Id      int
	Name    string
	Balance float32
}

// реализация скрыта, но там что-то вроде sqlx
type AccountRepository interface {
	Find(id int) (*Account, error)
	Save(acc *Account) error
}

// в реализации отправка в какой-то брокер, но может и быть http post
type EventSender interface {
	Send(event interface{}) error
}

type TransferEvent struct {
	FromId int
	ToId   int
	Amount float32
}

type AccountService struct {
	repository  AccountRepository
	eventSender EventSender
}

// вызывается из rest-контроллера
func (s *AccountService) Transfer(fromId int, toId int, amount float32) error {
	// Я сейчас буду ругаться
	// Во-первых мы должны выбрать аккаунты с локом на изменения под транзакцию
	// и соотв деферим ее закрытие
	fromAcc, err := s.repository.Find(fromId)
	if err != nil {
		return err
	}
	toAcc, err := s.repository.Find(toId)
	if err != nil {
		return err
	}
	// Во-вторых мы должны проверить есть ли там вообще достаточно денег?
	// if fromAcc.Balance < amount { return errors.New("not enough money") }

	toAcc.Balance += amount
	fromAcc.Balance -= amount

	err = s.repository.Save(fromAcc)
	if err != nil {
		return err
	} // Если ошибка, роллбек транзакции
	err = s.repository.Save(toAcc)
	if err != nil {
		return err
	}
	// Иначе, здесь коммит
	// Сохранять и изменять инстанции в данном случае лучше одновременно, поэтому надо перейти к изолированной транзакции (repeatable read)
	go func() {
		// ех, тут наверное по логике можно поретраить до первого успешного
		s.eventSender.Send(&TransferEvent{
			FromId: fromId,
			ToId:   toId,
			Amount: amount,
		})
	}()
	return nil
}
