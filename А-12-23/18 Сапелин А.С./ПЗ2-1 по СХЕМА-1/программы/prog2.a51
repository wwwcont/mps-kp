; Сапелин А.С., А-12-23, 18, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h
; здесь объявление переменных (если нужны)
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
org 1bh ; "заглушка" для Tmr1_ovf
        nop
        reti
org 23h ; "заглушка" для UART
        nop
        reti
org 2bh
START:
; Инициализация МК **********************************************
        mov SP, #07h
        ; TODO: CS_EN = 1; буфер пуст: HEAD = TAIL = BUF_START, F_EMPTY = 1, F_OVF = 0
        call BufRead ; %proc%
        jmp $ ; %stop%

; BufRead — чтение самого старого отсчёта из кольцевого буфера.
; Вход: нет. Выход: A — отсчёт; F_EMPTY — буфер опустел; F_OVF сброшен (буфер больше не полон).
; TODO: что делать при пустом буфере; что портит
BufRead:
        ; TODO: movx по адресу TAIL, TAIL на следующую ячейку (BUF_END → BUF_START), флаги
        ret
END
