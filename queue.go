package main

import (
	"errors"
	"sync"
)

// EstadoElemento equivale a los estados PENDIENTE y PROCESANDO del C.
type EstadoElemento int

const (
	PENDIENTE EstadoElemento = iota
	PROCESANDO
)

// Elemento representa un trabajo de la cola.
// En C la ruta era un char* asignado manualmente; en Go usamos string.
type Elemento struct {
	Ruta   string
	Estado EstadoElemento
}

// ErrColaVacia se devuelve cuando Dequeue se llama sin elementos.
var ErrColaVacia = errors.New("cola vacía")

// Cola es una cola FIFO segura para concurrencia.
// Sustituye a la cola circular con mutex global de C.
type Cola struct {
	mu        sync.Mutex
	cond      *sync.Cond
	elementos []Elemento
	dedupe    map[string]struct{}
	cerrada   bool
}

// NuevaCola crea una cola con la capacidad inicial indicada.
// El slice crecerá automáticamente, igual que el realloc del C.
func NuevaCola(capacidadInicial int) *Cola {
	if capacidadInicial < 1 {
		capacidadInicial = 1
	}
	q := &Cola{
		elementos: make([]Elemento, 0, capacidadInicial),
		dedupe:    make(map[string]struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue añade una ruta si no está ya en la cola.
// Si la cola fue cerrada, no hace nada.
func (q *Cola) Enqueue(ruta string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.cerrada {
		return
	}
	if _, existe := q.dedupe[ruta]; existe {
		return
	}
	q.elementos = append(q.elementos, Elemento{Ruta: ruta, Estado: PENDIENTE})
	q.dedupe[ruta] = struct{}{}
	q.cond.Signal()
}

// Dequeue extrae el primer elemento sin bloquear.
// Devuelve ErrColaVacia si no hay elementos disponibles.
func (q *Cola) Dequeue() (Elemento, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.elementos) == 0 {
		return Elemento{}, ErrColaVacia
	}
	elem := q.elementos[0]
	elem.Estado = PROCESANDO
	q.elementos[0] = Elemento{}
	q.elementos = q.elementos[1:]
	delete(q.dedupe, elem.Ruta)
	return elem, nil
}

// EsperarElemento bloquea hasta que exista un elemento.
// Devuelve false si la cola fue cerrada y está vacía.
func (q *Cola) EsperarElemento() (Elemento, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.elementos) == 0 && !q.cerrada {
		q.cond.Wait()
	}
	if len(q.elementos) == 0 {
		return Elemento{}, false
	}
	elem := q.elementos[0]
	elem.Estado = PROCESANDO
	q.elementos[0] = Elemento{}
	q.elementos = q.elementos[1:]
	delete(q.dedupe, elem.Ruta)
	return elem, true
}

// EsVacia informa si la cola no tiene elementos.
func (q *Cola) EsVacia() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.elementos) == 0
}

// Liberar vacía la cola, la cierra y despierta a todas las
// goroutines que estén esperando en EsperarElemento.
func (q *Cola) Liberar() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cerrada = true
	q.elementos = nil
	q.dedupe = make(map[string]struct{})
	q.cond.Broadcast()
}
