package main

import (
	"fmt"
	"sync"
	"time"
)

// Приложение эмулирует получение и обработку неких тасков. Пытается и получать, и обрабатывать в многопоточном режиме.
// Приложение должно генерировать таски 10 сек. Каждые 3 секунды должно выводить в консоль результат всех обработанных к этому моменту тасков (отдельно успешные и отдельно с ошибками).

// ЗАДАНИЕ: сделать из плохого кода хороший и рабочий - as best as you can.
// Важно сохранить логику появления ошибочных тасков.
// Важно оставить асинхронные генерацию и обработку тасков.
// Сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через pull-request в github
// Как видите, никаких привязок к внешним сервисам нет - полный карт-бланш на модификацию кода.

// A Ttype represents a meaninglessness of our life
type Ttype struct {
	id         int
	cT         string // время создания
	fT         string // время выполнения
	taskRESULT []byte
}

const (
	taskSuccess  = "task has been successed"
	taskError    = "something went wrong"
	errorOccured = "Some error occured"
)

func taskCreturer(a chan Ttype) {
	defer close(a)
	for start := time.Now(); time.Since(start) < time.Second*10; {
		ft := time.Now().Format(time.RFC3339)
		if time.Now().Nanosecond()%2 > 0 { // вот такое условие появления ошибочных тасков
			ft = errorOccured
		}
		a <- Ttype{cT: ft, id: int(time.Now().Unix())} // передаем таск на выполнение
	}
}

func task_worker(reciever, sortChan chan Ttype) {
	defer close(sortChan)
	for task := range reciever {
		tt, _ := time.Parse(time.RFC3339, task.cT)
		if tt.After(time.Now().Add(-20 * time.Second)) {
			task.taskRESULT = []byte(taskSuccess)
		} else {
			task.taskRESULT = []byte(taskError)
		}
		task.fT = time.Now().Format(time.RFC3339Nano)

		time.Sleep(time.Millisecond * 150)

		sortChan <- task
		if _, ok := <- reciever; !ok {break}
	}
}

func taskSorter(sortChan, doneTasks chan Ttype, undoneTasks chan error) {
	defer close(doneTasks)
	defer close(undoneTasks)
	for task := range sortChan {
		if string(task.taskRESULT) == taskSuccess {
			doneTasks <- task
		} else {
			undoneTasks <- fmt.Errorf("Task id %d time %s, error %s", task.id, task.cT, task.taskRESULT)
		}
		if _, ok := <- sortChan; !ok {break}
	}
}

func main() {
	superChan := make(chan Ttype, 10)
	go taskCreturer(superChan)

	sortChan := make(chan Ttype)
	go task_worker(superChan, sortChan)

	doneTasks := make(chan Ttype)
	undoneTasks := make(chan error)
	go taskSorter(sortChan, doneTasks, undoneTasks)

	result := map[int]Ttype{}
	err := []error{}

	var mutex sync.Mutex

	go func() {
		for task := range doneTasks {
			mutex.Lock()
			result[task.id] = task
			mutex.Unlock()
		}
	}()

	go func() {
		for er := range undoneTasks {
			mutex.Lock()
			err = append(err, er)
			mutex.Unlock()
		}
	}()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mutex.Lock()
		for key := range result {
			fmt.Printf("DONE	task ID: %d \n", key)
		}
		for _, v := range err {
			fmt.Printf("ERROR	%v \n", v)
		}
		mutex.Unlock()

		if _, ok := <-superChan; !ok {
			break
		}
	}
}
