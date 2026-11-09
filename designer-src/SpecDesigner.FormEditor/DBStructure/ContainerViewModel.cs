using System;
using System.ComponentModel;

namespace SpecDesigner.FormEditor.DBStructure
{
	// Token: 0x0200004A RID: 74
	public class ContainerViewModel : INotifyPropertyChanged
	{
		// Token: 0x17000089 RID: 137
		// (get) Token: 0x060002B3 RID: 691 RVA: 0x0000DEAA File Offset: 0x0000C0AA
		// (set) Token: 0x060002B4 RID: 692 RVA: 0x0000DEB2 File Offset: 0x0000C0B2
		public ContainerType Container { get; set; }

		// Token: 0x1700008A RID: 138
		// (get) Token: 0x060002B5 RID: 693 RVA: 0x0000DEBB File Offset: 0x0000C0BB
		// (set) Token: 0x060002B6 RID: 694 RVA: 0x0000DEC8 File Offset: 0x0000C0C8
		public string RepeatRowCountStrValue
		{
			get
			{
				return this._repeatRowCount.ToString();
			}
			set
			{
				this._repeatRowCount = 1;
				if (int.TryParse(value, out this._repeatRowCount))
				{
					this._repeatRowCount = Math.Max(this._repeatRowCount, 3 - this._repeatColumnCount);
				}
				this.NotifyPropertyChanged("RepeatRowCountStrValue");
			}
		}

		// Token: 0x1700008B RID: 139
		// (get) Token: 0x060002B7 RID: 695 RVA: 0x0000DF03 File Offset: 0x0000C103
		public int RepeatRowCountIntValue
		{
			get
			{
				return this._repeatRowCount;
			}
		}

		// Token: 0x1700008C RID: 140
		// (get) Token: 0x060002B8 RID: 696 RVA: 0x0000DF0B File Offset: 0x0000C10B
		// (set) Token: 0x060002B9 RID: 697 RVA: 0x0000DF18 File Offset: 0x0000C118
		public string RepeatColumnCountStrValue
		{
			get
			{
				return this._repeatColumnCount.ToString();
			}
			set
			{
				this._repeatColumnCount = 1;
				if (int.TryParse(value, out this._repeatColumnCount))
				{
					this._repeatColumnCount = Math.Max(this._repeatColumnCount, 3 - this._repeatRowCount);
				}
				this.NotifyPropertyChanged("RepeatColumnCountStrValue");
			}
		}

		// Token: 0x1700008D RID: 141
		// (get) Token: 0x060002BA RID: 698 RVA: 0x0000DF53 File Offset: 0x0000C153
		public int RepeatColumnCountIntValue
		{
			get
			{
				return this._repeatColumnCount;
			}
		}

		// Token: 0x1700008E RID: 142
		// (get) Token: 0x060002BB RID: 699 RVA: 0x0000DF5B File Offset: 0x0000C15B
		// (set) Token: 0x060002BC RID: 700 RVA: 0x0000DF68 File Offset: 0x0000C168
		public string MaximumWidthStrValue
		{
			get
			{
				return this._maximumWidth.ToString();
			}
			set
			{
				if (!int.TryParse(value, out this._maximumWidth))
				{
					this._maximumWidth = 30;
				}
				this.NotifyPropertyChanged("MaximumWidthStrValue");
			}
		}

		// Token: 0x1700008F RID: 143
		// (get) Token: 0x060002BD RID: 701 RVA: 0x0000DF8B File Offset: 0x0000C18B
		public int MaximumWidthIntValue
		{
			get
			{
				if (!int.TryParse(this.MaximumWidthStrValue, out this._maximumWidth))
				{
					this._maximumWidth = 30;
				}
				return this._maximumWidth;
			}
		}

		// Token: 0x17000090 RID: 144
		// (get) Token: 0x060002BE RID: 702 RVA: 0x0000DFAE File Offset: 0x0000C1AE
		// (set) Token: 0x060002BF RID: 703 RVA: 0x0000DFBB File Offset: 0x0000C1BB
		public string NumberOfFieldsStrValue
		{
			get
			{
				return this._numberOfFields.ToString();
			}
			set
			{
				if (!int.TryParse(value, out this._numberOfFields))
				{
					this._numberOfFields = 2;
				}
				this.NotifyPropertyChanged("NumberOfFieldsStrValue");
			}
		}

		// Token: 0x17000091 RID: 145
		// (get) Token: 0x060002C0 RID: 704 RVA: 0x0000DFDD File Offset: 0x0000C1DD
		public int NumberOfFieldsIntValue
		{
			get
			{
				if (!int.TryParse(this.NumberOfFieldsStrValue, out this._numberOfFields))
				{
					this._numberOfFields = 2;
				}
				return this._numberOfFields;
			}
		}

		// Token: 0x1400000A RID: 10
		// (add) Token: 0x060002C1 RID: 705 RVA: 0x0000E000 File Offset: 0x0000C200
		// (remove) Token: 0x060002C2 RID: 706 RVA: 0x0000E038 File Offset: 0x0000C238
		public event PropertyChangedEventHandler PropertyChanged;

		// Token: 0x060002C3 RID: 707 RVA: 0x0000E06D File Offset: 0x0000C26D
		private void NotifyPropertyChanged(string property)
		{
			if (this.PropertyChanged != null)
			{
				this.PropertyChanged(this, new PropertyChangedEventArgs(property));
			}
		}

		// Token: 0x04000184 RID: 388
		private int _repeatRowCount = 2;

		// Token: 0x04000185 RID: 389
		private int _repeatColumnCount = 1;

		// Token: 0x04000186 RID: 390
		private int _numberOfFields = 2;

		// Token: 0x04000187 RID: 391
		private int _maximumWidth = 30;
	}
}
