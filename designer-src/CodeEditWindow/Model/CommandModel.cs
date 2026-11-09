using System;
using System.Collections.Generic;
using System.Windows.Input;

namespace SpecDesigner.CodeEditWindow.Model
{
	// Token: 0x0200002D RID: 45
	public class CommandModel
	{
		// Token: 0x1700003A RID: 58
		// (get) Token: 0x060001B8 RID: 440 RVA: 0x0000F281 File Offset: 0x0000D481
		// (set) Token: 0x060001B9 RID: 441 RVA: 0x0000F289 File Offset: 0x0000D489
		public ICommand Command { get; set; }

		// Token: 0x1700003B RID: 59
		// (get) Token: 0x060001BA RID: 442 RVA: 0x0000F292 File Offset: 0x0000D492
		// (set) Token: 0x060001BB RID: 443 RVA: 0x0000F29A File Offset: 0x0000D49A
		public object Parameter { get; set; }

		// Token: 0x1700003C RID: 60
		// (get) Token: 0x060001BC RID: 444 RVA: 0x0000F2A3 File Offset: 0x0000D4A3
		// (set) Token: 0x060001BD RID: 445 RVA: 0x0000F2AB File Offset: 0x0000D4AB
		public string Text { get; set; }

		// Token: 0x1700003D RID: 61
		// (get) Token: 0x060001BE RID: 446 RVA: 0x0000F2B4 File Offset: 0x0000D4B4
		// (set) Token: 0x060001BF RID: 447 RVA: 0x0000F2BC File Offset: 0x0000D4BC
		public bool IsChecked
		{
			get
			{
				return this._isChecked;
			}
			set
			{
				this._isChecked = value;
			}
		}

		// Token: 0x1700003E RID: 62
		// (get) Token: 0x060001C0 RID: 448 RVA: 0x0000F2C5 File Offset: 0x0000D4C5
		// (set) Token: 0x060001C1 RID: 449 RVA: 0x0000F2CD File Offset: 0x0000D4CD
		public bool IsCheckable { get; set; }

		// Token: 0x1700003F RID: 63
		// (get) Token: 0x060001C2 RID: 450 RVA: 0x0000F2D6 File Offset: 0x0000D4D6
		// (set) Token: 0x060001C3 RID: 451 RVA: 0x0000F2DE File Offset: 0x0000D4DE
		public List<CommandModel> Children { get; set; }

		// Token: 0x060001C4 RID: 452 RVA: 0x0000F2E7 File Offset: 0x0000D4E7
		public CommandModel()
		{
			this.Parameter = string.Empty;
			this.Text = string.Empty;
			this.IsCheckable = false;
			this.IsChecked = false;
		}

		// Token: 0x060001C5 RID: 453 RVA: 0x0000F313 File Offset: 0x0000D513
		public CommandModel(ICommand command, string text)
			: this()
		{
			this.Command = command;
			this.Text = text;
		}

		// Token: 0x040000BF RID: 191
		private bool _isChecked;
	}
}
