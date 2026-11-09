using System;
using System.Collections.Generic;
using SpecDesignerCommon.ViewModel;
using UndoRedoFramework.Commands;

namespace SpecDesignerCommon.UndoRedo
{
	// Token: 0x02000031 RID: 49
	public class MultiHideUndoRedoCommand : IUndoRedoCommand
	{
		// Token: 0x06000176 RID: 374 RVA: 0x00007948 File Offset: 0x00005B48
		public MultiHideUndoRedoCommand(IEnumerable<XmlElement> elements, bool newValue)
		{
			this._records = new Dictionary<XmlElement, BooleanAttributeValue>();
			foreach (XmlElement xmlElement in elements)
			{
				if (null == this._key)
				{
					this._key = xmlElement.Key;
				}
				BooleanAttributeValue booleanAttributeValue = new BooleanAttributeValue();
				booleanAttributeValue.OldValue = xmlElement.IsHidden;
				booleanAttributeValue.NewValue = newValue;
				this._records.Add(xmlElement, booleanAttributeValue);
			}
		}

		// Token: 0x06000177 RID: 375 RVA: 0x000079DC File Offset: 0x00005BDC
		public void Execute()
		{
			if (this._records.Count == 0)
			{
				return;
			}
			foreach (KeyValuePair<XmlElement, BooleanAttributeValue> keyValuePair in this._records)
			{
				XmlElement key = keyValuePair.Key;
				BooleanAttributeValue value = keyValuePair.Value;
				key.IsBatchSet = true;
				key.IsHidden = value.NewValue;
				key.OnPropertyChanged("hidden");
				key.IsBatchSet = false;
			}
		}

		// Token: 0x06000178 RID: 376 RVA: 0x00007A6C File Offset: 0x00005C6C
		public void Undo()
		{
			if (this._records.Count == 0)
			{
				return;
			}
			foreach (KeyValuePair<XmlElement, BooleanAttributeValue> keyValuePair in this._records)
			{
				XmlElement key = keyValuePair.Key;
				BooleanAttributeValue value = keyValuePair.Value;
				key.IsBatchSet = true;
				key.IsHidden = value.OldValue;
				key.OnPropertyChanged("hidden");
				key.IsBatchSet = false;
			}
		}

		// Token: 0x06000179 RID: 377 RVA: 0x00007AFC File Offset: 0x00005CFC
		public void Clear()
		{
			if (this._records != null)
			{
				this._records.Clear();
			}
		}

		// Token: 0x04000096 RID: 150
		private Dictionary<XmlElement, BooleanAttributeValue> _records;

		// Token: 0x04000097 RID: 151
		private PackageKey _key;
	}
}
