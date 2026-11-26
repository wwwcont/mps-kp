; Панов Н.М., А-17-23, 11, v1
$INCLUDE (vars.inc)             ; адреса, биты и константы варианта
DSEG AT 30h
X1:     DS 1                    ; цифра с клавиатуры (0 — Y2 пассивен)
X2:     DS 1                    ; отсчёт из буфера
CSEG AT 0h
        sjmp START
org 03h ; "заглушка" для INT0
        nop
        reti
org 0bh ; "заглушка" для Tmr0_ovf
        nop
        reti
org 13h ; "заглушка" для INT1
        nop
        reti
org 1bh                         ; Timer1 — истекло время строба Y2
        clr TR1                 ; останавливаем таймер
        setb PIN_Y2             ; строб снова пассивен (1)
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        setb PIN_CSEN           ; дешифратор CS включён
        setb PIN_Y2             ; строб Y2 пока не активен
        anl TMOD, #0Fh          ; сохраняем режим Timer0
        orl TMOD, #10h          ; Timer1 — 16-битный режим 1
        setb ET1                ; прерывание по переполнению Timer1
        setb EA                 ; разрешаем прерывания
        mov X1, #3              ; пример входных данных
        mov X2, #200
        call Y2Out ; %proc%
        jmp $ ; %stop%

; Y2Out — расчёт и выдача Y2 = (G+M+X1+X2) mod 256; при X1=0 Y2=0.
; Вход: X1, X2 в ОЗУ. Выход: регистр Y2; PIN_Y2 = 0 на T2 мкс
; (окончание строба выполняет обработчик Timer1).
; Портит: A, DPTR, PSW (сохраняются в стеке).
Y2Out:
        push PSW
        push ACC
        push DPL
        push DPH
        mov A, X1
        jz Y2Zero               ; X1 = 0 — пассивное состояние
        mov A, #Y2_CONST        ; G + M
        add A, X1
        add A, X2               ; перенос отбрасывается: результат mod 256
        sjmp Y2Write
Y2Zero:
        clr A                   ; Y2 = 0
Y2Write:
        mov DPTR, #ADR_Y2
        movx @DPTR, A           ; сначала данные в регистр Y2
        clr TR1                 ; перезапуск, если прошлый строб ещё шёл
        mov TH1, #T2_RELOAD_H
        mov TL1, #T2_RELOAD_L
        clr TF1                 ; старый запрос переполнения не нужен
        clr PIN_Y2              ; начало строба
        setb TR1
        pop DPH
        pop DPL
        pop ACC
        pop PSW
        ret
END
