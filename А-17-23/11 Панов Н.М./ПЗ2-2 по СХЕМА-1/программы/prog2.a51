; Панов Н.М., А-17-23, 11, v1
$INCLUDE (vars.inc)             ; адреса, биты и константы варианта
DSEG AT 30h
; отдельных переменных нет: указатели хранятся в ОЗУ по именам из vars.inc
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
        mov SP, #07h            ; стек ниже банков 1–3
        setb PIN_CSEN           ; дешифратор CS включён
        mov HEAD_L, #LOW(BUF_START)     ; голова — начало буфера
        mov HEAD_H, #HIGH(BUF_START)
        mov TAIL_L, #LOW(BUF_START)     ; хвост — тоже начало
        mov TAIL_H, #HIGH(BUF_START)
        setb F_EMPTY            ; буфер пуст
        clr F_OVF               ; переполнения ещё не было
        mov A, #55h             ; пример отсчёта X2
        call BufWrite ; %proc%
        jmp $ ; %stop%

; BufWrite — запись отсчёта в кольцевой буфер.
; Вход: A — отсчёт. Выход: F_EMPTY = 0; F_OVF = 1, если запись затёрла самый старый
; отсчёт (буфер был полон). Портит: A, DPTR, PSW; R2 сохраняется в стеке.
BufWrite:
        push PSW                ; сохраняем флаги вызывающего
        push DPL                ; и его DPTR
        push DPH
        push AR2                ; R2 используем как временный буфер
        mov R2, A               ; сохраняем отсчёт на время проверки полноты
        jb F_EMPTY, BwStore     ; пустой буфер — место точно есть
        mov A, HEAD_L           ; сравниваем младшие байты указателей
        cjne A, TAIL_L, BwStore
        mov A, HEAD_H           ; затем старшие
        cjne A, TAIL_H, BwStore
        ; голова догнала хвост при непустом буфере — буфер полон:
        ; выбрасываем самый старый отсчёт, сдвигая хвост
        mov DPL, TAIL_L         ; DPTR = текущий хвост
        mov DPH, TAIL_H
        call BufNext            ; хвост на следующий отсчёт
        mov TAIL_L, DPL         ; сохраняем новый хвост
        mov TAIL_H, DPH
        setb F_OVF              ; старый отсчёт потерян — переполнение
BwStore:
        mov DPL, HEAD_L         ; DPTR = текущая голова
        mov DPH, HEAD_H
        mov A, R2               ; восстанавливаем записываемый отсчёт
        movx @DPTR, A           ; запись отсчёта в ячейку «головы»
        call BufNext            ; голова на следующую ячейку
        mov HEAD_L, DPL         ; сохраняем новую голову
        mov HEAD_H, DPH
        clr F_EMPTY             ; после записи буфер не пуст
        pop AR2                 ; восстанавливаем регистры вызывающего
        pop DPH
        pop DPL
        pop PSW
        ret

; BufNext — переводит DPTR на следующую ячейку буфера.
; После BUF_END−1 идёт BUF_START (сравниваются оба байта адреса).
; Вход/выход: DPTR. Портит: A.
BufNext:
        inc DPTR                ; следующая ячейка
        mov A, DPL              ; младший байт равен границе?
        cjne A, #LOW(BUF_END), BnDone
        mov A, DPH              ; проверяем старший байт
        cjne A, #HIGH(BUF_END), BnDone
        mov DPTR, #BUF_START    ; заворот в начало буфера
BnDone:
        ret
END
