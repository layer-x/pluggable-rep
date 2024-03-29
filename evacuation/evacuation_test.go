package evacuation_test

import (
	"errors"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/executor"
	"github.com/cloudfoundry-incubator/executor/fakes"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry-incubator/rep/evacuation"
	"github.com/cloudfoundry-incubator/rep/evacuation/evacuation_context"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation", func() {
	const (
		cellID            = "cell-id"
		pollingInterval   = 30 * time.Second
		evacuationTimeout = time.Duration(6) * pollingInterval
	)

	var (
		logger             *lagertest.TestLogger
		fakeClock          *fakeclock.FakeClock
		executorClient     *fakes.FakeClient
		evacuatable        evacuation_context.Evacuatable
		evacuationNotifier evacuation_context.EvacuationNotifier

		evacuator *evacuation.Evacuator
		process   ifrit.Process

		errChan chan error

		TaskTags   map[string]string
		LRPTags    map[string]string
		containers []executor.Container
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")
		fakeClock = fakeclock.NewFakeClock(time.Now())
		executorClient = &fakes.FakeClient{}

		evacuatable, _, evacuationNotifier = evacuation_context.New()

		evacuator = evacuation.NewEvacuator(
			logger,
			fakeClock,
			executorClient,
			evacuationNotifier,
			cellID,
			evacuationTimeout,
			pollingInterval,
		)

		process = ifrit.Invoke(evacuator)

		errChan = make(chan error, 1)

		localErrChan := errChan
		evacuationProcess := process
		go func() {
			localErrChan <- <-evacuationProcess.Wait()
		}()

		TaskTags = map[string]string{rep.LifecycleTag: rep.TaskLifecycle}
		LRPTags = map[string]string{
			rep.LifecycleTag:    rep.LRPLifecycle,
			rep.DomainTag:       "domain",
			rep.ProcessGuidTag:  "process-guid",
			rep.ProcessIndexTag: "2",
		}
		containers = []executor.Container{
			{Guid: "guid-1", State: executor.StateRunning, Tags: TaskTags},
			{Guid: "guid-2", State: executor.StateRunning, Tags: LRPTags},
		}
	})

	Describe("before evacuating", func() {
		It("exits when interrupted", func() {
			process.Signal(os.Interrupt)

			Eventually(errChan).Should(Receive(BeNil()))
		})
	})

	Describe("during evacuation", func() {
		JustBeforeEach(func() {
			evacuatable.Evacuate()
		})

		Context("when containers are present", func() {
			Context("and are all destroyed before the timeout elapses", func() {
				BeforeEach(func() {
					containerResponses := [][]executor.Container{
						containers,
						[]executor.Container{},
					}

					index := 0
					executorClient.ListContainersStub = func(lager.Logger) ([]executor.Container, error) {
						containersToReturn := containerResponses[index]
						index++
						return containersToReturn, nil
					}
				})

				It("waits for all the containers to go away and exits before evacuation timeout", func() {
					fakeClock.Increment(pollingInterval)
					Eventually(executorClient.ListContainersCallCount).Should(Equal(1))

					fakeClock.Increment(pollingInterval)
					Eventually(executorClient.ListContainersCallCount).Should(Equal(2))

					Eventually(errChan).Should(Receive(BeNil()))
				})

				Context("when the executor client returns an error", func() {
					BeforeEach(func() {
						index := 0
						executorClient.ListContainersStub = func(lager.Logger) ([]executor.Container, error) {
							if index == 0 {
								index++
								return nil, errors.New("whoops")
							}
							return []executor.Container{}, nil
						}
					})

					It("retries", func() {
						fakeClock.Increment(pollingInterval)
						Eventually(executorClient.ListContainersCallCount).Should(Equal(1))

						fakeClock.Increment(pollingInterval)
						Eventually(executorClient.ListContainersCallCount).Should(Equal(2))

						Eventually(errChan).Should(Receive(BeNil()))
					})
				})
			})

			Context("and are not all destroyed before the timeout elapses", func() {
				BeforeEach(func() {
					executorClient.ListContainersReturns(containers, nil)
				})

				It("exits after the evacuation timeout", func() {
					Eventually(fakeClock.WatcherCount).Should(Equal(2))

					fakeClock.Increment(evacuationTimeout - time.Second)
					Consistently(errChan).ShouldNot(Receive())
					fakeClock.Increment(2 * time.Second)
					Eventually(errChan).Should(Receive(BeNil()))
				})

				Context("when signaled", func() {
					It("exits", func() {
						process.Signal(os.Interrupt)

						Eventually(errChan).Should(Receive(BeNil()))
					})
				})
			})
		})
	})
})
