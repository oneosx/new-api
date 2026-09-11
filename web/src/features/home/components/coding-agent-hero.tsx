/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Link } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'
import { DanaoBrainIcon } from '@/assets/danao-brain'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export function CodingAgentHero() {
  const { t } = useTranslation()
  const [hoveredModel, setHoveredModel] = useState<string | null>(null)

  return (
    <div className='relative w-full max-w-7xl mx-auto px-2 my-16'>
      <div className='relative z-10 grid grid-cols-1 lg:grid-cols-12 gap-4 items-center'>
        {/* Right Column: 3D Galaxy Planetary System (Scaled to prevent clipping) */}
        <div className='order-1 lg:order-2 lg:col-span-6 relative flex items-center justify-center w-full select-none overflow-visible'>
          <div className='scale-75 md:scale-120 h-[300px] md:h-[500px] solar-system-stage origin-center transition-transform duration-300'>
            
            {/* Tilted Space Disc (Carries 3D dashed concentric orbit circles) */}
            <div className='-mt-[150px] md:-mt-[250px] -ml-[160px] md:-ml-[260px] solar-system-plane'>
              {/* Orbit 1: Claude (Diameter: 340px, Radius: 170px) */}
              <div className='solar-orbit-ring solar-ring-1' />
              {/* Orbit 2: OpenAI (Diameter: 460px, Radius: 230px) */}
              <div className='solar-orbit-ring solar-ring-2' />
              {/* Orbit 3: Gemini (Diameter: 580px, Radius: 290px) */}
              <div className='solar-orbit-ring solar-ring-3' />
              {/* Orbit 4: Grok (Diameter: 700px, Radius: 350px) */}
              <div className='solar-orbit-ring solar-ring-4' />

              {/* Central Brain: Converted from danao.eps vector SVG */}
              <div
                className='absolute top-13/24 left-13/24 -translate-x-1/2 -translate-y-1/2 z-15 flex items-center justify-center pointer-events-none'
                style={{
                  transform: 'translate(-50%, -50%) rotateX(-68deg) rotateZ(30deg)',
                }}
              >
                <DanaoBrainIcon className='w-12 h-10 sm:w-14 sm:h-12 drop-shadow-md select-none' />
              </div>

              {/* =========================================================
                  4 REVOLVING PLANETS (True 3D Upright Billboarding)
                 ========================================================= */}

              {/* Planet 1: Claude (Orbit 1) */}
              <div
                className='solar-planet-body'
                style={{
                  color: '#d97757',
                  animation: 'solar-orbit-claude 16s linear infinite',
                }}
              >
                <svg className='size-4 sm:size-4.5' viewBox='0 0 24 24' fill='currentColor'>
                  <path d='m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z' />
                </svg>
              </div>

              {/* Planet 2: OpenAI (Orbit 2) */}
              <div
                className='solar-planet-body'
                style={{
                  color: '#10a37f',
                  animation: 'solar-orbit-openai 22s linear infinite',
                }}
              >
                <svg className='size-5 sm:size-5.5' viewBox='0 0 24 24' fill='currentColor'>
                  <path d='M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855l-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.666zm2.01-3.023l-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135l-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681zm1.097-2.365l2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z' />
                </svg>
              </div>

              {/* Planet 3: Gemini (Orbit 3) */}
              <div
                className='solar-planet-body'
                style={{
                  color: '#2aa198',
                  animation: 'solar-orbit-gemini 28s linear infinite',
                }}
              >
                <svg className='size-4 sm:size-4.5' viewBox='0 0 24 24' fill='currentColor'>
                  <path d='M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81' />
                </svg>
              </div>

              {/* Planet 4: Grok (Orbit 4) */}
              <div
                className='solar-planet-body'
                style={{
                  color: '#050505',
                  animation: 'solar-orbit-grok 34s linear infinite',
                }}
              >
                <svg className='size-4 sm:size-4.5' viewBox='0 0 512 512' fill='currentColor'>
                  <path d='M210.484 312.759L343.465 210.383C349.984 205.364 359.302 207.322 362.408 215.117C378.758 256.231 371.454 305.64 338.925 339.563C306.397 373.487 261.137 380.927 219.768 363.983L174.577 385.803C239.394 432.008 318.104 420.581 367.289 369.251C406.303 328.564 418.386 273.104 407.088 223.091L407.19 223.198C390.807 149.726 411.218 120.359 453.03 60.3072C454.02 58.8833 455.01 57.4595 456 56L400.978 113.382V113.204L210.45 312.794' />
                  <path d='M183.042 337.641C136.519 291.294 144.54 219.567 184.236 178.203C213.59 147.59 261.683 135.096 303.666 153.464L348.755 131.75C340.632 125.627 330.221 119.042 318.275 114.414C264.277 91.2407 199.63 102.774 155.735 148.516C113.513 192.549 100.236 260.254 123.036 318.027C140.069 361.206 112.148 391.748 84.0229 422.575C74.0561 433.503 64.0553 444.431 56 456L183.007 337.677' />
                </svg>
              </div>
            </div>

          </div>
        </div>

        {/* Mobile: Typography & Action on BOTTOM (order-2 lg:order-1), Desktop: Left Column */}
        <div className='order-2 lg:order-1 lg:col-span-6 flex flex-col items-center lg:items-start text-center lg:text-left mt-2 lg:mt-0'>
          <h1 className='text-3xl sm:text-5xl lg:text-[3.25rem] font-bold tracking-tight text-foreground leading-[1.18]'>
            {t('home.hero.title')}
          </h1>
          <p className='mt-4 max-w-xl text-sm sm:text-lg text-muted-foreground leading-relaxed'>
            {t('home.hero.subtitle')}
          </p>
          <div className='mt-6 lg:mt-8'>
            <Link
              to='/dashboard'
              className='group inline-flex items-center gap-2 rounded-full bg-foreground px-6 py-3 text-sm font-medium text-background transition-all duration-200 hover:opacity-90 hover:shadow-lg hover:shadow-foreground/10 active:scale-[0.98]'
            >
              <span>{t('home.hero.cta')}</span>
              <ChevronRight className='size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}
