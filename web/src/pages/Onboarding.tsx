"use client"

import { useState } from "react"
import { useNavigate } from "react-router"

import { ConnectStep } from "@/components/onboarding/ConnectStep"
import { FirstRunStep } from "@/components/onboarding/FirstRunStep"
import { MobileSteps, ObBar } from "@/components/onboarding/ObBar"
import { WelcomeStep } from "@/components/onboarding/WelcomeStep"
import { readOnboarding, saveOnboarding } from "@/lib/onboarding"

/** The first-run flow at /welcome: welcome, connect, first run. Skippable, and it resumes where it stopped. */
export function Onboarding() {
  const navigate = useNavigate()
  const [state, setState] = useState(readOnboarding)

  const go = (step: number) => {
    saveOnboarding({ step })
    setState((now) => ({ ...now, step }))
  }
  const setFolder = (folder: string) => {
    saveOnboarding({ folder })
    setState((now) => ({ ...now, folder }))
  }
  const setAgent = (agent: string) => {
    saveOnboarding({ agent })
    setState((now) => ({ ...now, agent }))
  }
  const skip = () => {
    saveOnboarding({ skipped: true })
    navigate("/")
  }
  const started = (id: string) => {
    saveOnboarding({ done: true })
    navigate(`/spans/${encodeURIComponent(id)}`)
  }

  return (
    <div className="flex min-h-dvh flex-col bg-background">
      <ObBar step={state.step} onSkip={skip} />
      <main className="flex w-full flex-1 flex-col px-4 pt-6 pb-10 sm:px-10 sm:pt-12 sm:pb-16">
        <div className="mb-6 sm:hidden">
          <MobileSteps step={state.step} />
        </div>
        {state.step === 1 ? (
          <WelcomeStep onNext={() => go(2)} onSkip={skip} />
        ) : null}
        {state.step === 2 ? (
          <ConnectStep
            folder={state.folder}
            agent={state.agent}
            savedAt={state.savedAt}
            onFolder={setFolder}
            onAgent={setAgent}
            onNext={() => go(3)}
            onBack={() => go(1)}
          />
        ) : null}
        {state.step === 3 ? (
          <FirstRunStep
            folder={state.folder}
            agent={state.agent}
            onStarted={started}
            onBack={() => go(2)}
            onSkip={skip}
          />
        ) : null}
      </main>
    </div>
  )
}
