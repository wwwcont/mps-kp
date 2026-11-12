; Сапелин А.С., А-12-23, 18, v1
$INCLUDE (vars.inc)
DSEG AT 30h
CSEG AT 0h
        sjmp START
org 03h
        call KbScan ; %proc%
        reti
org 0bh
        nop
        reti
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
        mov SP, #07h            ; стек размещается ниже переменных и битовых флагов
        mov PSW, #0             ; используется нулевой банк регистров
        setb PIN_CSEN           ; разрешить выбор устройств внешней шины
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; все столбцы готовы обнаружить новое нажатие
        setb IT0                ; однократный запрос по спаду INT0
        clr IE0                 ; исключить запрос от начальной установки столбцов
        setb EX0
        setb EA
        jmp $ ; %stop%

; Назначение: KbScan проверяет все контакты матрицы и считает нажатые клавиши.
; Вход: строки и столбцы клавиатуры, подключённой к ADR_KB.
; Выход: A — код 0–11 единственной клавиши либо 0FFh при ином числе нажатий.
; Портит: A; остальные используемые регистры сохраняются.
KbScan:
        push PSW
        push DPL
        push DPH
        push AR0
        push AR1
        push AR2
        push AR3
        push AR4
        push AR5
        mov R0, #0FEh           ; бегущий ноль начинает со столбца Col1
        mov R1, #0              ; индекс первого контакта столбца в таблице
        mov R2, #0              ; пока ни одна клавиша не обнаружена
        mov R3, #0FFh
KbColumn:
        mov DPTR, #ADR_KB
        mov A, R0
        movx @DPTR, A           ; выбрать один столбец
        movx A, @DPTR
        cpl A
        anl A, #(1 SHL KB_ROWS)-1 ; оставить только реально подключённые строки
        mov R4, A
        mov R5, #KB_ROWS
KbRow:
        mov A, R4
        rrc A                   ; очередная строка попадает в перенос
        mov R4, A
        jnc KbNextRow
        inc R2                  ; считаются все замкнутые контакты
        mov DPTR, #KeyMap
        mov A, R1
        movc A, @A+DPTR
        mov R3, A
KbNextRow:
        inc R1
        djnz R5, KbRow
        mov A, R0
        rl A
        mov R0, A               ; перенести активный ноль на следующий столбец
        cjne R1, #(KB_COLS*KB_ROWS), KbColumn
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; вернуть все столбцы в состояние ожидания
        clr IE0                 ; убрать фронты, возникшие во время сканирования
        mov A, R2
        cjne A, #1, KbInvalid
        mov A, R3
        sjmp KbDone
KbInvalid:
        mov A, #0FFh
KbDone:
        pop AR5
        pop AR4
        pop AR3
        pop AR2
        pop AR1
        pop AR0
        pop DPH
        pop DPL
        pop PSW
        ret
KeyMap:
        db 1,4,7,2,5,8,3,6,9,10,11,0
END
