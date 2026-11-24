# {Título}

## Historia

Como: {persona en una situación}
Quiero: {capacidad}
Para: {resultado que esa persona reconoce}

## Contexto de Uso

Disparo: {evento que produce esta historia}
Qué se mantiene: {comportamiento que esta historia no cambia}
Qué cambia: {delta que esta historia posee}
Datos y origen de la verdad: {cada dato visible y de dónde sale}

## Alcance

Incluye:

- {resultado que esta historia posee}

No incluye:

- {vecino que el prompt toca y esta historia no cambia}

## Reglas de negocio

BR100 {una oración declarativa, una sola restricción}

## Criterios de Aceptación

AC 100 - {nombre}
{condición observable, verdadera o falsa}
Confirma: BR100

## Escenarios de prueba

| ID | Escenario de Prueba | Contexto (Dado que…) | Acción (Cuando…) | Resultado Esperado (Entonces…) | Cubre |
| --- | --- | --- | --- | --- | --- |
| TC-01 | {ejemplo clave} | Dado que {datos concretos} | Cuando {acción} | Entonces {resultado observable con esos datos} | AC 100 |

## Impacto

| Superficie | Efecto | Evidencia |
| --- | --- | --- |
| {superficie nombrada en el prompt} | {qué cambia junto con esta historia} | Observado: {qué dijo el prompt} |

## Supuestos

- {Ninguno, o un hecho usado que el prompt no afirmó}
