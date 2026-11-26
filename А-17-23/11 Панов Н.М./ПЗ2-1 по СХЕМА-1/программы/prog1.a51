; Панов Н.М., А-17-23, 11, v1
$INCLUDE (vars.inc)             ; адреса, биты и константы варианта
DSEG AT 30h
; отдельных переменных нет: состояние опроса держим в регистрах банка 0
CSEG AT 0h
        sjmp START
org 03h                         ; INT0 — нажатие клавиши: обработчик вызывает опрос
        call KbScan ; %proc%
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
        setb PIN_CSEN           ; дешифратор CS включён
        mov DPTR, #ADR_KB
        clr A
        movx @DPTR, A           ; все столбцы опущены: нажатие сразу даст INT0
        setb IT0                ; INT0 — по спаду
        clr IE0                 ; сбрасываем возможный старый запрос
        setb EX0                ; разрешаем прерывание клавиатуры
        setb EA                 ; разрешаем прерывания
        jmp $ ; %stop%

; KbScan — опрос матрицы клавиатуры 3x4 по столбцам.
; Вход: нет. Выход: A — код единственной нажатой клавиши (0…9, Т=10, С=11)
; или 0FFh, если нажато ничего либо две и более клавиш.
; Портит: A, B, DPTR, PSW; регистры R2–R7 восстанавливаются из стека.
KbScan:
        push PSW
        push B
        push DPL
        push DPH
        push AR2
        push AR3
        push AR4
        push AR5
        push AR6
        push AR7
        mov DPTR, #ADR_KB
        mov R2, #0              ; число найденных нажатий
        mov R3, #0FFh           ; код последней найденной клавиши
        mov R4, #0FEh           ; маска столбцов: 0 — опрашиваемый
        mov R5, #0              ; номер столбца 0…KB_COLS-1
KsCol:
        mov A, R4
        movx @DPTR, A           ; опускаем только текущий столбец
        movx A, @DPTR           ; читаем строки
        anl A, #(1 SHL KB_ROWS) - 1     ; старшие биты шины не подключены
        xrl A, #(1 SHL KB_ROWS) - 1     ; 1 — строка замкнута
        mov R6, A               ; битовая карта строк
        mov R7, #0              ; номер строки
KsRow:
        mov A, R6
        anl A, #1               ; проверяем младшую (текущую) строку
        jz KsRowNext
        inc R2                  ; нашли ещё одно нажатие
        cjne R7, #KB_ROWS - 1, KsCalc
        ; четвёртая строка: клавиши 0, Т, С
        mov A, R5
        jz KsZero
        cjne A, #1, KsTest
        mov A, #10              ; «Т»
        sjmp KsSave
KsZero:
        clr A                   ; «0»
        sjmp KsSave
KsTest:
        mov A, #11              ; «С»
        sjmp KsSave
KsCalc:
        ; строки 1–3: код = номер_строки*KB_COLS + номер_столбца + 1
        mov A, R7
        mov B, #KB_COLS
        mul AB
        add A, R5
        inc A
KsSave:
        mov R3, A
KsRowNext:
        mov A, R6
        rr A                    ; переходим к следующей строке
        mov R6, A
        inc R7
        cjne R7, #KB_ROWS, KsRow
        mov A, R4
        rl A                    ; следующий столбец
        mov R4, A
        inc R5
        cjne R5, #KB_COLS, KsCol
        clr A
        movx @DPTR, A           ; после опроса все столбцы снова опущены
        mov A, R3
        cjne R2, #1, KsBad      ; ровно одно нажатие — отдаём код
        sjmp KsDone
KsBad:
        mov A, #0FFh            ; ничего или несколько клавиш
KsDone:
        clr IE0                 ; линия INT0 дёргалась при переборе столбцов
        pop AR7
        pop AR6
        pop AR5
        pop AR4
        pop AR3
        pop AR2
        pop DPH
        pop DPL
        pop B
        pop PSW
        ret
END
