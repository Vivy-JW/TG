using System;
using System.ComponentModel;

namespace SpecDesigner.FormEditor.DBStructure
{
	// Token: 0x02000029 RID: 41
	public class PrepareAddColumn : INotifyPropertyChanged
	{
		// Token: 0x14000004 RID: 4
		// (add) Token: 0x06000178 RID: 376 RVA: 0x000085F4 File Offset: 0x000067F4
		// (remove) Token: 0x06000179 RID: 377 RVA: 0x0000862C File Offset: 0x0000682C
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x1700004B RID: 75
		// (get) Token: 0x0600017A RID: 378 RVA: 0x00008661 File Offset: 0x00006861
		// (set) Token: 0x0600017B RID: 379 RVA: 0x00008669 File Offset: 0x00006869
		public string Column { get; set; }

		// Token: 0x1700004C RID: 76
		// (get) Token: 0x0600017C RID: 380 RVA: 0x00008672 File Offset: 0x00006872
		// (set) Token: 0x0600017D RID: 381 RVA: 0x0000867A File Offset: 0x0000687A
		public string Description { get; set; }

		// Token: 0x1700004D RID: 77
		// (get) Token: 0x0600017E RID: 382 RVA: 0x00008683 File Offset: 0x00006883
		// (set) Token: 0x0600017F RID: 383 RVA: 0x0000868B File Offset: 0x0000688B
		public bool IsSelected
		{
			get
			{
				return this._isSelected;
			}
			set
			{
				this._isSelected = value;
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs("IsSelected"));
				}
			}
		}

		// Token: 0x1700004E RID: 78
		// (get) Token: 0x06000180 RID: 384 RVA: 0x000086B2 File Offset: 0x000068B2
		// (set) Token: 0x06000181 RID: 385 RVA: 0x000086D8 File Offset: 0x000068D8
		public string Label
		{
			get
			{
				if (this._label == null)
				{
					this._label = string.Format("lbl_{0}", this.Column);
				}
				return this._label;
			}
			set
			{
				this._label = value;
				if (this.PropertyChanged != null)
				{
					this.PropertyChanged(this, new PropertyChangedEventArgs("Label"));
				}
			}
		}

		// Token: 0x1700004F RID: 79
		// (get) Token: 0x06000182 RID: 386 RVA: 0x000086FF File Offset: 0x000068FF
		// (set) Token: 0x06000183 RID: 387 RVA: 0x00008707 File Offset: 0x00006907
		public string Table { get; set; }

		// Token: 0x17000050 RID: 80
		// (get) Token: 0x06000184 RID: 388 RVA: 0x00008710 File Offset: 0x00006910
		// (set) Token: 0x06000185 RID: 389 RVA: 0x00008718 File Offset: 0x00006918
		public string Widget { get; set; }

		// Token: 0x17000051 RID: 81
		// (get) Token: 0x06000186 RID: 390 RVA: 0x00008721 File Offset: 0x00006921
		// (set) Token: 0x06000187 RID: 391 RVA: 0x00008729 File Offset: 0x00006929
		public string Width { get; set; }

		// Token: 0x17000052 RID: 82
		// (get) Token: 0x06000188 RID: 392 RVA: 0x00008734 File Offset: 0x00006934
		public int WidthIntValue
		{
			get
			{
				int num = 0;
				if (int.TryParse(this.Width, out num))
				{
					return num;
				}
				return 10;
			}
		}

		// Token: 0x040000DB RID: 219
		private bool _isSelected;

		// Token: 0x040000DC RID: 220
		private string _label;
	}
}
