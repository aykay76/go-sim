package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type SimulationState struct {
	CurrentPrice        int
	TotalBookings       int
	MaximumBots         int
	ActiveBots          []string
	ChanceOfBooking     int
	NumberOfLowerPrices int
}
type BotState struct {
	Lifetime    int
	LiveSteps   int
	State       string
	Booking     bool
	LowestPrice int
	LastPrice   int
	TimedOut    bool
}

type Simulation struct {
	ActualStartTime  time.Time
	ActualFinishTime time.Time
	StartTime        time.Time
	CurrentTime      time.Time
	TimeStep         time.Duration
	AutoStep         bool
	DoneChan         chan struct{}
	Mutex            sync.Mutex
	RNG              *rand.Rand
	StepsTaken       int
	EndSteps         int
	ActorCount       int
	CurrentActors    int
	State            SimulationState
}

func NewSimulation() *Simulation {
	return &Simulation{
		AutoStep:        true,
		ActualStartTime: time.Now(),
		StartTime:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		CurrentTime:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		TimeStep:        time.Second,
		DoneChan:        make(chan struct{}),
		RNG:             rand.New(rand.NewSource(99)),
		EndSteps:        1800,
		ActorCount:      1,
		CurrentActors:   0,
		State: SimulationState{
			CurrentPrice:        4000,
			TotalBookings:       0,
			ChanceOfBooking:     100,
			MaximumBots:         5,
			NumberOfLowerPrices: 0,
		},
	}
}

func removeString(slice []string, s string) []string {
	for i, v := range slice {
		if v == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func main() {
	sim := NewSimulation()
	sim.StepsTaken = 0
	sim.ActualStartTime = time.Now()

	RNG := rand.New(rand.NewSource(99))

	// create a population of bots
	bots := make(map[string]*BotState)
	for i := 1; i < 18000; i++ {
		bots[fmt.Sprintf("bot%d", i)] = &BotState{
			Lifetime:    0,
			LiveSteps:   0,
			State:       "dormant",
			Booking:     false,
			LowestPrice: 0,
			LastPrice:   0,
			TimedOut:    false,
		}
	}

	for sim.StepsTaken < sim.EndSteps {
		prevActiveBots := len(sim.State.ActiveBots)

		// make enough bots active
		for i := 1; i < 18000 && len(sim.State.ActiveBots) < prevActiveBots+sim.State.MaximumBots; i++ {
			id := fmt.Sprintf("bot%d", i)
			bot := bots[id]
			if bot.State == "dormant" {
				bot.State = "active"
				bot.LiveSteps = 0
				bot.Booking = false
				bot.TimedOut = false
				bot.LowestPrice = -1
				bot.LastPrice = 0
				if RNG.Intn(100)+1 < sim.State.ChanceOfBooking {
					bot.Lifetime = RNG.Intn(235) + 6
					bot.Booking = true
					bot.TimedOut = false
				} else {
					bot.Booking = false
					if RNG.Intn(2)+1 == 1 {
						bot.Lifetime = 600
						bot.TimedOut = true
					} else {
						bot.Lifetime = RNG.Intn(595) + 6
					}
				}
				sim.State.ActiveBots = append(sim.State.ActiveBots, id)

				fmt.Println("🏃‍♂️", id, "Actor is now active, booking:", bot.Booking, "timed out:", bot.TimedOut)
			}
		}

		// step the active bots
		for _, id := range sim.State.ActiveBots {
			bot := bots[id]
			if bot.Booking {
				if bot.LiveSteps >= bot.Lifetime {
					fmt.Println("🧡", id, "Actor is booking at price:", sim.State.CurrentPrice)
					if bot.LowestPrice == -1 {
						bot.LowestPrice = sim.State.CurrentPrice
					} else {
						if sim.State.CurrentPrice < bot.LowestPrice {
							bot.LowestPrice = sim.State.CurrentPrice
							sim.State.NumberOfLowerPrices++
						}
					}
					// done, booked
					sim.State.TotalBookings++
					bot.State = "dormant"
					// remove from active bots
					sim.State.ActiveBots = removeString(sim.State.ActiveBots, id)

					if sim.State.TotalBookings%20 == 0 {
						sim.State.CurrentPrice += 5
						sim.State.ChanceOfBooking--
						if sim.State.ChanceOfBooking < 0 {
							sim.State.ChanceOfBooking = 0
						}
						fmt.Println("☝🏻 Price increased to: ", sim.State.CurrentPrice, ", chance of booking: ", sim.State.ChanceOfBooking)
					}
				}
			} else {
				if bot.LiveSteps >= bot.Lifetime {
					fmt.Println("🤕", id, "Actor exceeded lifetime (timeout or drop off)")
					bot.State = "dormant"
					sim.State.ActiveBots = removeString(sim.State.ActiveBots, id)
				}
			}

			bot.LiveSteps++
		}

		sim.StepsTaken++
		sim.CurrentTime = sim.CurrentTime.Add(sim.TimeStep)
		fmt.Println("Step:", sim.StepsTaken, ", simulation time:", sim.CurrentTime, "; Active bots:", len(sim.State.ActiveBots))
	}

	fmt.Println("🛑 Simulation state: Active bots=", len(sim.State.ActiveBots), "Chance of booking=", sim.State.ChanceOfBooking, "Current price=", sim.State.CurrentPrice, "Lower prices=", sim.State.NumberOfLowerPrices, "Total bookings=", sim.State.TotalBookings)
}
