package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type SimulationState struct {
}
type BotState struct {
	GotLowerPrice    bool
	ActivePrice      int
	PreviouslyActive bool
	Lifetime         int
	LiveSteps        int
	State            string
	Booking          bool
	LowestPrice      int
	TimedOut         bool
	SearchPrice      int
	BookingPrice     int
}

type Simulation struct {
	ActualStartTime     time.Time
	ActualFinishTime    time.Time
	StartTime           time.Time
	CurrentTime         time.Time
	TimeStep            time.Duration
	AutoStep            bool
	DoneChan            chan struct{}
	Mutex               sync.Mutex
	RNG                 *rand.Rand
	StepsTaken          int
	EndSteps            int
	ActorCount          int
	CurrentActors       int
	CurrentPrice        int
	TotalBookings       int
	MaximumBots         int
	ActiveBots          []string
	ChanceOfBooking     int
	NumberOfLowerPrices int
	AvailablePrices     []int
	TotalActivations    int
	HigherPriceLocked   int
}

func NewSimulation() *Simulation {
	return &Simulation{
		AutoStep:            true,
		ActualStartTime:     time.Now(),
		StartTime:           time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		CurrentTime:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		TimeStep:            time.Second,
		DoneChan:            make(chan struct{}),
		RNG:                 rand.New(rand.NewSource(99)),
		EndSteps:            1800,
		ActorCount:          1,
		CurrentActors:       0,
		CurrentPrice:        4000,
		TotalBookings:       0,
		ChanceOfBooking:     100,
		MaximumBots:         42,
		NumberOfLowerPrices: 0,
		TotalActivations:    0,
		HigherPriceLocked:   0,
	}
}

func removeInt(slice []int, val int) []int {
	for i, v := range slice {
		if v == val {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
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

	RNG := rand.New(rand.NewSource(time.Now().UnixNano()))

	populationSize := 18000

	// create a population of bots
	bots := make(map[string]*BotState)
	for i := 1; i < populationSize; i++ {
		bots[fmt.Sprintf("bot%d", i)] = &BotState{
			Lifetime:         0,
			LiveSteps:        0,
			State:            "dormant",
			Booking:          false,
			LowestPrice:      0,
			TimedOut:         false,
			PreviouslyActive: false,
		}
	}

	for sim.StepsTaken < sim.EndSteps {
		prevActiveBots := len(sim.ActiveBots)

		// make enough bots active
		for i := 1; i < populationSize && len(sim.ActiveBots) < prevActiveBots+sim.MaximumBots; i++ {
			id := fmt.Sprintf("bot%d", i)
			bot := bots[id]
			if bot.State == "dormant" {
				bot.State = "active"
				sim.TotalActivations++

				// take a price from the available set, if any
				if len(sim.AvailablePrices) > 0 {
					bot.GotLowerPrice = true
					bot.ActivePrice = sim.AvailablePrices[0]
					sim.AvailablePrices = removeInt(sim.AvailablePrices, bot.ActivePrice)
				} else {
					bot.GotLowerPrice = false
					bot.ActivePrice = sim.CurrentPrice
					bot.SearchPrice = sim.CurrentPrice
				}

				bot.LiveSteps = 0
				bot.Booking = false
				bot.TimedOut = false
				bot.LowestPrice = -1
				if RNG.Intn(100)+1 < sim.ChanceOfBooking {
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
				sim.ActiveBots = append(sim.ActiveBots, id)

				if bot.PreviouslyActive {
					fmt.Println("♻️", id, "Actor is active again, booking:", bot.Booking, "timed out:", bot.TimedOut, "price quoted:", bot.ActivePrice)
				} else {
					bot.PreviouslyActive = true
					fmt.Println("🏃", id, "Actor is now active, booking:", bot.Booking, "timed out:", bot.TimedOut, "price quoted:", bot.ActivePrice)
				}
			}
		}

		// step the active bots
		for _, id := range sim.ActiveBots {
			bot := bots[id]
			if bot.Booking {
				if bot.LiveSteps == 1 {
					// lock in the price
					if sim.CurrentPrice > bot.SearchPrice {
						bot.ActivePrice = sim.CurrentPrice
						sim.HigherPriceLocked++
					}
				}

				if bot.LiveSteps >= bot.Lifetime {
					if bot.GotLowerPrice {
						fmt.Println("💖", id, "New booking at lower price:", bot.ActivePrice)
						sim.NumberOfLowerPrices++
						bot.LowestPrice = bot.ActivePrice
					} else {
						fmt.Println("💗", id, "Booking at current market price:", bot.ActivePrice)
						bot.LowestPrice = bot.ActivePrice
					}

					// done, booked
					sim.TotalBookings++
					bot.State = "dormant"
					// remove from active bots
					sim.ActiveBots = removeString(sim.ActiveBots, id)

					if sim.TotalBookings%20 == 0 {
						sim.CurrentPrice += 5
						sim.ChanceOfBooking--
						if sim.ChanceOfBooking < 1 {
							sim.ChanceOfBooking = 1
						}
						fmt.Println("☝🏻 Price increased to: ", sim.CurrentPrice, ", chance of booking: ", sim.ChanceOfBooking)
					}
				}
			} else {
				if bot.LiveSteps >= bot.Lifetime {
					fmt.Println("🤕", id, "Actor exceeded lifetime (timeout or drop off), will return availability at price:", bot.ActivePrice)
					bot.State = "dormant"
					sim.ActiveBots = removeString(sim.ActiveBots, id)
					sim.AvailablePrices = append(sim.AvailablePrices, bot.ActivePrice)
				}
			}

			bot.LiveSteps++
		}

		sim.StepsTaken++
		sim.CurrentTime = sim.CurrentTime.Add(sim.TimeStep)
		fmt.Println("Step =", sim.StepsTaken, ", simulation time =", sim.CurrentTime, "; Active bots =", len(sim.ActiveBots))
	}

	fmt.Println("🛑 Simulation state: Active bots =", len(sim.ActiveBots), "Chance of booking =", sim.ChanceOfBooking, "Current price =", sim.CurrentPrice, "Lower prices =", sim.NumberOfLowerPrices, "Total bookings =", sim.TotalBookings, "Total activations =", sim.TotalActivations, "Higher price locked =", sim.HigherPriceLocked)
}
