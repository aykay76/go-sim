package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type BotState struct {
	GotLowerPrice    bool
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
	StepsTaken          int
	EndSteps            int
	CurrentPrice        int
	TotalBookings       int
	MaximumBots         int
	ActiveBots          []string
	ChanceOfBooking     int
	NumberOfLowerPrices int
	AvailablePrices     []int
	TotalSearches       int
	HigherPriceLocked   int
	LockedCurrentPrice  int
}

func NewSimulation() *Simulation {
	return &Simulation{
		AutoStep:            true,
		ActualStartTime:     time.Now(),
		StartTime:           time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		CurrentTime:         time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		TimeStep:            time.Second,
		DoneChan:            make(chan struct{}),
		EndSteps:            1800,
		CurrentPrice:        4000,
		TotalBookings:       0,
		ChanceOfBooking:     100,
		MaximumBots:         42,
		NumberOfLowerPrices: 0,
		TotalSearches:       0,
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
		bots[fmt.Sprintf("bot-%d", i)] = &BotState{
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
		fmt.Println("Step =", sim.StepsTaken, ", simulation time =", sim.CurrentTime, "; Active bots =", len(sim.ActiveBots))

		prevActiveBots := len(sim.ActiveBots)

		// make enough bots active
		for i := 1; i < populationSize && len(sim.ActiveBots) < prevActiveBots+sim.MaximumBots; i++ {
			id := fmt.Sprintf("bot-%d", i)
			bot := bots[id]
			if bot.State == "dormant" {
				bot.State = "active"
				sim.TotalSearches++

				// take a price from the available set, if any
				if len(sim.AvailablePrices) > 0 {
					bot.GotLowerPrice = true
					bot.SearchPrice = sim.AvailablePrices[0]
					sim.AvailablePrices = removeInt(sim.AvailablePrices, bot.SearchPrice)
				} else {
					bot.GotLowerPrice = false
					bot.SearchPrice = sim.CurrentPrice
				}

				bot.LiveSteps = 0
				bot.Booking = false
				bot.TimedOut = false
				bot.LowestPrice = -1

				chanceOfBooking := 100 - ((bot.SearchPrice - 4000) / 5)

				if RNG.Intn(100)+1 < chanceOfBooking {
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
					fmt.Println("♻️", id, "is active again, booking:", bot.Booking, "timed out:", bot.TimedOut, "search price:", bot.SearchPrice)
				} else {
					bot.PreviouslyActive = true
					fmt.Println("🏃", id, "is now active, booking:", bot.Booking, "timed out:", bot.TimedOut, "search price:", bot.SearchPrice)
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
						bot.SearchPrice = sim.CurrentPrice
						sim.HigherPriceLocked++
					}

					fmt.Println("🔒", id, "locked price at:", sim.CurrentPrice)

					if !bot.GotLowerPrice {
						sim.LockedCurrentPrice++
						if sim.LockedCurrentPrice == 20 {
							sim.CurrentPrice += 5
							sim.LockedCurrentPrice = 0

							sim.ChanceOfBooking--
							if sim.ChanceOfBooking < 0 {
								sim.ChanceOfBooking = 0
							}
							fmt.Println("🔏 Price increased to: ", sim.CurrentPrice, "chance of booking: ", sim.ChanceOfBooking)
						}
					}
				}

				if bot.LiveSteps >= bot.Lifetime {
					if bot.GotLowerPrice {
						fmt.Println("💖", id, "booked at lower price:", bot.SearchPrice)
						sim.NumberOfLowerPrices++
						bot.LowestPrice = bot.SearchPrice
					} else {
						fmt.Println("💗", id, "booked at current market price:", bot.SearchPrice)
						bot.LowestPrice = bot.SearchPrice
					}

					// done, booked
					sim.TotalBookings++
					bot.State = "dormant"
					// remove from active bots
					sim.ActiveBots = removeString(sim.ActiveBots, id)

					if sim.TotalBookings%20 == 0 {
						sim.CurrentPrice += 5
						sim.ChanceOfBooking--
						if sim.ChanceOfBooking < 0 {
							sim.ChanceOfBooking = 0
						}
						fmt.Println("☝🏻 Price increased to: ", sim.CurrentPrice, ", chance of booking: ", sim.ChanceOfBooking)
					}
				}
			} else {
				if bot.LiveSteps >= bot.Lifetime {
					fmt.Println("🤕", id, "exceeded lifetime (timeout or drop off), will return availability at price:", bot.SearchPrice)
					bot.State = "dormant"
					sim.ActiveBots = removeString(sim.ActiveBots, id)
					sim.AvailablePrices = append(sim.AvailablePrices, bot.SearchPrice)
				}
			}

			bot.LiveSteps++
		}

		sim.StepsTaken++
		sim.CurrentTime = sim.CurrentTime.Add(sim.TimeStep)
	}

	fmt.Println("🛑 Simulation state: Active bots =", len(sim.ActiveBots), "Current price =", sim.CurrentPrice, "Lower prices =", sim.NumberOfLowerPrices, "Total bookings =", sim.TotalBookings, "Total searches =", sim.TotalSearches, "Higher price locked =", sim.HigherPriceLocked)
}
