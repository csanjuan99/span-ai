"use client"

import { useState } from "react"
import { useNavigate } from "react-router"

import { ConnectStep } from "@/components/onboarding/ConnectStep"
import { FirstRunStep } from "@/components/onboarding/FirstRunStep"
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
  const skip = () => {
    saveOnboarding({ skipped: true })
    navigate("/")
  }
  const start = () => {
    saveOnboarding({ done: true })
    navigate("/", { state: { sample: true } })
  }

  return (
    <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-10">
      {state.step === 1 ? (
        <WelcomeStep onNext={() => go(2)} onSkip={skip} />
      ) : null}
      {state.step === 2 ? (
        <ConnectStep
          folder={state.folder}
          onFolder={setFolder}
          onNext={() => go(3)}
          onBack={() => go(1)}
          onSkip={skip}
        />
      ) : null}
      {state.step === 3 ? (
        <FirstRunStep
          folder={state.folder}
          onStart={start}
          onBack={() => go(2)}
          onSkip={skip}
        />
      ) : null}
    </main>
  )
}
