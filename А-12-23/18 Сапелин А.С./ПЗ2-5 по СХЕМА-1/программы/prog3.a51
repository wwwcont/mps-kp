; Сапелин А.С., А-12-23, 18, v1
$INCLUDE (vars.inc)
DSEG AT 30h
TicksLo: DS 1
TicksHi: DS 1
CSEG AT 0h
        sjmp START
org 03h
        nop
        reti
org 0bh
        ljmp T0Isr
org 13h
        nop
        reti
org 1bh
        nop
        reti
org 23h
        nop
        reti
org 2bh
START:
        mov SP, #07h            ; переменные 30h и выше не пересекаются со стеком
        mov IE, #0
        setb PIN_CSEN           ; разрешить запись в регистр индикации
        clr TR0
        clr TF0                 ; удалить прежний запрос переполнения
        mov TMOD, #01h          ; Timer0 считает 16-разрядные интервалы
        mov TicksLo, #0
        mov TicksHi, #0          ; до первого вызова отсчёт не ведётся
        mov DPTR, #ADR_IND
        mov A, #IND_OFF
        movx @DPTR, A           ; при включении все сегменты погашены
        setb ET0
        setb EA
        mov A, #5               ; демонстрационный вывод цифры 5
        call Show ; %proc%
        jmp $ ; %stop%

; Назначение: Show выводит символ с автоматическим гашением через T3.
; Вход: A — код 0–9 или 10 для E; остальные коды гасят индикатор.
; Выход: символ в регистре ADR_IND, заново запущенный отсчёт T3 либо остановка.
; Портит: A, DPTR, PSW; обновляет TicksLo, TicksHi и Timer0.
Show:
        clr ET0                 ; исключить IRQ со старым счётчиком во время замены символа
        clr TR0
        clr TF0                 ; повторный вызов отменяет предыдущий срок
        mov TicksLo, #0
        mov TicksHi, #0
        cjne A, #11, ShowRange
        sjmp ShowOff
ShowRange:
        jnc ShowOff             ; код за границей таблицы означает выключение
        mov DPTR, #Segments
        movc A, @A+DPTR          ; код символа преобразуется в набор сегментов
        xrl A, #IND_OFF          ; общий анод требует активного нуля
        mov DPTR, #ADR_IND
        movx @DPTR, A           ; регистр удержит изображение после освобождения шины
        mov TicksLo, #LOW T3_TICKS
        mov TicksHi, #HIGH T3_TICKS ; число тиков берётся из параметров варианта
        mov TH0, #TICK_H
        mov TL0, #TICK_L         ; первый тик начинается с полной длительности
        setb ET0
        setb TR0                ; процедура возвращается, время отсчитывается в IRQ
        ret
ShowOff:
        mov DPTR, #ADR_IND
        mov A, #IND_OFF
        movx @DPTR, A           ; неверный код сразу выключает все сегменты
        setb ET0
        ret

; Назначение: T0Isr отсчитывает тики и выключает индикатор в конце T3.
; Вход: TicksLo:TicksHi и текущее состояние Timer0.
; Выход: счётчик уменьшен; при нуле Timer0 остановлен и индикатор погашен.
; Портит: Timer0 и счётчик; A, PSW и DPTR восстанавливаются перед RETI.
T0Isr:
        push ACC                ; обработчик может прервать работу с любым символом
        push PSW
        push DPL
        push DPH                ; сохранить адрес устройства, использованный основной программой
        clr TR0
        mov A, TL0              ; учесть время между переполнением и входом в обработчик
        add A, #(TICK_L+7)      ; компенсировать семь циклов остановленного таймера
        mov TL0, A
        mov A, TH0
        addc A, #TICK_H
        mov TH0, A
        setb TR0                ; задержка обслуживания не накапливается в каждом тике
        mov A, TicksLo
        jnz TickDec
        dec TicksHi             ; заём при уменьшении 16-разрядного счётчика
TickDec:
        dec TicksLo
        mov A, TicksLo
        orl A, TicksHi
        jnz TickDone            ; ненулевой остаток сохраняет текущую индикацию
        clr TR0
        mov DPTR, #ADR_IND
        mov A, #IND_OFF
        movx @DPTR, A           ; срок T3 истёк
TickDone:
        pop DPH
        pop DPL
        pop PSW
        pop ACC
        reti
Segments:
        db 03Fh,006h,05Bh,04Fh,066h,06Dh,07Dh,007h,07Fh,06Fh,079h
END
