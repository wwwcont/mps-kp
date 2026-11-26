; Хромов И., А-08-23, 20, v1
$INCLUDE (vars.inc)             ; адреса, биты, константы варианта; в файл для робота mpscode вклеит его текст
DSEG AT 30h

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
        setb PIN_CSEN ; разрешить доступ к внешнему ОЗУ
        mov DPTR, #BUF_START ; пустой буфер начинается с первого адреса
        mov HEAD_L, DPL
        mov HEAD_H, DPH
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        setb F_EMPTY ; при совпадении адресов буфер опустел
        clr F_OVF ; после чтения появилась свободная ячейка
        call BufRead ; %proc%
        jmp $ ; %stop%

; BufRead — чтение самого старого отсчёта из кольцевого буфера.
; Вход: нет. Выход: A — отсчёт; F_EMPTY — буфер опустел; F_OVF сброшен (буфер больше не полон).
; Для пустого буфера A = 0, флаги и указатели сохраняются. Изменяет DPTR и CY.
BufRead:
        jb F_EMPTY, Empty ; при пустом буфере чтения нет
        mov DPL, TAIL_L ; загрузить адрес старейшего отсчёта
        mov DPH, TAIL_H
        movx A, @DPTR ; прочитать отсчёт из левого порта ОЗУ
        push ACC ; сохранить отсчёт на время расчёта адреса
        inc DPTR ; перейти к следующей ячейке
        mov A, DPL
        cjne A, #LOW BUF_END, SaveTail ; сначала проверить младший байт границы
        mov A, DPH
        cjne A, #HIGH BUF_END, SaveTail ; сравнить старший байт границы
        mov DPTR, #BUF_START ; пустой буфер начинается с первого адреса
SaveTail:
        mov TAIL_L, DPL
        mov TAIL_H, DPH
        clr F_OVF ; после чтения появилась свободная ячейка
        mov A, DPL
        cjne A, HEAD_L, ReadDone ; сравнить новый хвост с головой
        mov A, DPH
        cjne A, HEAD_H, ReadDone
        setb F_EMPTY ; при совпадении адресов буфер опустел
ReadDone:
        pop ACC ; вернуть прочитанный отсчёт
        ret
Empty:
        clr A
        ret
END
